// Package custommgr registers user-defined (custom) providers into the live
// registry, prefix maps, and display catalog — on boot and on change.
package custommgr

import (
	"context"

	"github.com/enowdev/enowx/core/provider"
	"github.com/enowdev/enowx/core/provider/compat"
	"github.com/enowdev/enowx/core/proxy"
	"github.com/enowdev/enowx/store"
)

// Catalog is the subset of the providers-catalog the manager mutates (satisfied
// by server/handlers via function values, to avoid an import cycle).
type Catalog struct {
	Add    func(name, label, icon string)
	Remove func(name string)
}

type Manager struct {
	reg   *provider.Registry
	store store.CustomProviderStore
	cat   Catalog
}

func New(reg *provider.Registry, s store.CustomProviderStore, cat Catalog) *Manager {
	return &Manager{reg: reg, store: s, cat: cat}
}

// RegisterOne registers an already-persisted provider live (used by sync when a
// custom provider is pulled from the cloud).
func (m *Manager) RegisterOne(p store.CustomProvider) { m.register(p) }

// UnregisterOne removes a provider from the registry/prefix/catalog by
// prefix+name (used by sync on a pulled deletion).
func (m *Manager) UnregisterOne(prefix, name string) {
	m.reg.Unregister(name)
	proxy.RemovePrefix(prefix, name)
	if m.cat.Remove != nil {
		m.cat.Remove(name)
	}
}

// LoadAll registers every stored custom provider (called on boot).
func (m *Manager) LoadAll(ctx context.Context) error {
	list, err := m.store.List(ctx)
	if err != nil {
		return err
	}
	for _, p := range list {
		m.register(p)
	}
	return nil
}

// register wires one definition into the registry + prefix + catalog.
func (m *Manager) register(p store.CustomProvider) {
	models := make([]provider.Model, 0, len(p.Models))
	for _, mm := range p.Models {
		name := mm.Name
		if name == "" {
			name = mm.ID
		}
		models = append(models, provider.Model{ID: mm.ID, Name: name, Type: "chat"})
	}
	m.reg.Register(compat.New(compat.Def{Name: p.Name, Format: p.Format, BaseURL: p.BaseURL, Models: models}))
	proxy.AddPrefix(p.Prefix, p.Name)
	if m.cat.Add != nil {
		m.cat.Add(p.Name, p.Name, "custom")
	}
}

// unregister removes one custom provider live.
func (m *Manager) unregister(p store.CustomProvider) {
	m.reg.Unregister(p.Name)
	proxy.RemovePrefix(p.Prefix, p.Name)
	if m.cat.Remove != nil {
		m.cat.Remove(p.Name)
	}
}

// Add persists + registers a new custom provider.
func (m *Manager) Add(ctx context.Context, p store.CustomProvider) (int64, error) {
	id, err := m.store.Create(ctx, p)
	if err != nil {
		return 0, err
	}
	p.ID = id
	m.register(p)
	return id, nil
}

// Update persists changes + re-registers (unregister old prefix, register new).
func (m *Manager) Update(ctx context.Context, p store.CustomProvider) error {
	old, err := m.store.Get(ctx, p.ID)
	if err != nil {
		return err
	}
	if err := m.store.Update(ctx, p); err != nil {
		return err
	}
	if old != nil {
		m.unregister(*old)
	}
	m.register(p)
	return nil
}

// Remove deletes + unregisters a custom provider.
func (m *Manager) Remove(ctx context.Context, id int64) error {
	old, err := m.store.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := m.store.Delete(ctx, id); err != nil {
		return err
	}
	if old != nil {
		m.unregister(*old)
	}
	return nil
}

// List returns the stored definitions.
func (m *Manager) List(ctx context.Context) ([]store.CustomProvider, error) {
	return m.store.List(ctx)
}

// Probe fetches models from an upstream (for the add-provider form preview).
func (m *Manager) Probe(baseURL, format, apiKey string) ([]provider.Model, error) {
	p := compat.New(compat.Def{Name: "probe", Format: format, BaseURL: baseURL})
	return p.Models(provider.Account{Creds: map[string]string{"api_key": apiKey}})
}
