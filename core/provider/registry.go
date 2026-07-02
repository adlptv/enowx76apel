package provider

import (
	"fmt"
	"sync"
)

type Registry struct {
	mu sync.RWMutex
	m  map[string]Provider
}

func NewRegistry() *Registry { return &Registry{m: map[string]Provider{}} }

func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	r.m[p.Name()] = p
	r.mu.Unlock()
}

// Unregister removes a provider (used when a custom provider is deleted).
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	delete(r.m, name)
	r.mu.Unlock()
}

func (r *Registry) Get(name string) (Provider, error) {
	r.mu.RLock()
	p, ok := r.m[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("provider %q not registered", name)
	}
	return p, nil
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.m))
	for n := range r.m {
		out = append(out, n)
	}
	return out
}
