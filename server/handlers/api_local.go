package handlers

import (
	"net/http"

	"github.com/enowdev/enowx/core/localcreds"
	"github.com/enowdev/enowx/store"
)

// Local imports accounts from credentials that installed IDEs/CLIs wrote to disk.
type Local struct {
	store  store.AccountStore
	warmer Warmer
}

func NewLocal(s store.AccountStore) *Local { return &Local{store: s} }

// SetWarmer enables automatic warmup of imported accounts.
func (h *Local) SetWarmer(w Warmer) { h.warmer = w }

type localSourceDTO struct {
	Provider string `json:"provider"`
	Target   string `json:"target"`
	Path     string `json:"path"`
}

func (h *Local) Scan(w http.ResponseWriter, _ *http.Request) {
	found := localcreds.Scan()
	out := make([]localSourceDTO, 0, len(found))
	for _, f := range found {
		out = append(out, localSourceDTO{Provider: f.Provider, Target: f.Target, Path: f.Path})
	}
	writeData(w, out)
}

// POST /api/local-sources/import { "provider": "kiro", "target": "Kiro Desktop" }
func (h *Local) Import(w http.ResponseWriter, r *http.Request) {
	var in struct{ Provider, Target string }
	readJSON(r, &in)
	f, ok := localcreds.Get(in.Provider, in.Target)
	if !ok {
		writeAPIErr(w, http.StatusNotFound, "no local credentials found for that source")
		return
	}
	id, err := h.store.Add(r.Context(), store.Account{
		Provider: f.Provider,
		Label:    f.Target,
		Creds:    f.Creds,
		Status:   "active",
	})
	if err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := map[string]any{"id": id}
	if warm := autoWarm(r.Context(), h.warmer, h.store, id); warm != nil {
		out["warmup"] = warm
	}
	writeData(w, out)
}
