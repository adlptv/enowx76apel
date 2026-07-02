package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/enowdev/enowx/core/provider/custommgr"
	"github.com/enowdev/enowx/core/proxy"
	"github.com/enowdev/enowx/server/middleware"
	"github.com/enowdev/enowx/store"
)

// CustomProviders manages user-added OpenAI/Anthropic-compatible providers.
type CustomProviders struct {
	dash *middleware.Dashboard
	mgr  *custommgr.Manager
	acct store.AccountStore
}

func NewCustomProviders(dash *middleware.Dashboard, mgr *custommgr.Manager, acct store.AccountStore) *CustomProviders {
	return &CustomProviders{dash: dash, mgr: mgr, acct: acct}
}

func (h *CustomProviders) guard(w http.ResponseWriter, r *http.Request) bool {
	if !h.dash.Authorized(r) {
		writeAPIErr(w, http.StatusForbidden, "requires the dashboard login when accessed remotely")
		return false
	}
	return true
}

var prefixRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,23}$`)

type customProviderBody struct {
	Name         string              `json:"name"`
	Prefix       string              `json:"prefix"`
	Format       string              `json:"format"`
	BaseURL      string              `json:"base_url"`
	DefaultModel string              `json:"default_model"`
	Models       []store.CustomModel `json:"models"`
	APIKey       string              `json:"api_key"` // becomes the provider's first account
}

func (b customProviderBody) toStore() store.CustomProvider {
	format := b.Format
	if format != "anthropic" {
		format = "openai"
	}
	if b.Models == nil {
		b.Models = []store.CustomModel{}
	}
	return store.CustomProvider{
		Name: strings.TrimSpace(b.Name), Prefix: strings.ToLower(strings.TrimSpace(b.Prefix)),
		Format: format, BaseURL: strings.TrimRight(strings.TrimSpace(b.BaseURL), "/"),
		DefaultModel: strings.TrimSpace(b.DefaultModel), Models: b.Models,
	}
}

func (h *CustomProviders) List(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w, r) {
		return
	}
	list, err := h.mgr.List(r.Context())
	if err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, map[string]any{"providers": list})
}

func (h *CustomProviders) Create(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w, r) {
		return
	}
	var body customProviderBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIErr(w, http.StatusBadRequest, "bad body")
		return
	}
	p := body.toStore()
	if p.Name == "" || p.BaseURL == "" {
		writeAPIErr(w, http.StatusBadRequest, "name and base URL are required")
		return
	}
	if !prefixRe.MatchString(p.Prefix) {
		writeAPIErr(w, http.StatusBadRequest, "prefix must be lowercase letters/numbers/dashes")
		return
	}
	if proxy.PrefixTaken(p.Prefix) {
		writeAPIErr(w, http.StatusConflict, "prefix is already in use")
		return
	}
	id, err := h.mgr.Add(r.Context(), p)
	if err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	// The api key entered when adding the provider becomes its first account.
	if key := strings.TrimSpace(body.APIKey); key != "" && h.acct != nil {
		_, _ = h.acct.Add(r.Context(), store.Account{
			Provider: p.Name,
			Label:    p.Name + " key",
			Creds:    map[string]string{"api_key": key},
			Status:   "active",
		})
	}
	writeData(w, map[string]any{"id": id})
}

func (h *CustomProviders) Update(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w, r) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body customProviderBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIErr(w, http.StatusBadRequest, "bad body")
		return
	}
	p := body.toStore()
	p.ID = id
	if !prefixRe.MatchString(p.Prefix) {
		writeAPIErr(w, http.StatusBadRequest, "invalid prefix")
		return
	}
	// Allow the same prefix on update; reject only if taken by a different provider.
	if proxy.PrefixTaken(p.Prefix) {
		if pv, _ := proxy.SplitModel(p.Prefix + "/x"); pv != "" && pv != p.Name {
			writeAPIErr(w, http.StatusConflict, "prefix is already in use")
			return
		}
	}
	if err := h.mgr.Update(r.Context(), p); err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, map[string]any{"ok": true})
}

func (h *CustomProviders) Delete(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w, r) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	// Find the provider name so we can also delete its accounts (cascade).
	name := ""
	if list, err := h.mgr.List(r.Context()); err == nil {
		for _, p := range list {
			if p.ID == id {
				name = p.Name
				break
			}
		}
	}
	if err := h.mgr.Remove(r.Context(), id); err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Delete the provider's accounts too — they're unusable without it.
	if name != "" && h.acct != nil {
		if accs, err := h.acct.List(r.Context(), name); err == nil {
			for _, a := range accs {
				_ = h.acct.Delete(r.Context(), a.ID)
			}
		}
	}
	writeData(w, map[string]any{"ok": true})
}

// Probe fetches models from an upstream so the form can preview/auto-fill.
func (h *CustomProviders) Probe(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w, r) {
		return
	}
	var body struct {
		BaseURL string `json:"base_url"`
		Format  string `json:"format"`
		APIKey  string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIErr(w, http.StatusBadRequest, "bad body")
		return
	}
	models, err := h.mgr.Probe(strings.TrimRight(strings.TrimSpace(body.BaseURL), "/"), body.Format, strings.TrimSpace(body.APIKey))
	if err != nil {
		writeAPIErr(w, http.StatusBadGateway, err.Error())
		return
	}
	out := make([]store.CustomModel, 0, len(models))
	for _, m := range models {
		out = append(out, store.CustomModel{ID: m.ID, Name: m.Name})
	}
	writeData(w, map[string]any{"models": out})
}
