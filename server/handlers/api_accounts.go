package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/enowdev/enowx/store"
)

// Accounts is the management API over the credential pool.
type Accounts struct{ store store.AccountStore }

func NewAccounts(s store.AccountStore) *Accounts { return &Accounts{store: s} }

type accountDTO struct {
	ID        int64    `json:"id"`
	Provider  string   `json:"provider"`
	Label     string   `json:"label"`
	Status    string   `json:"status"`
	Has       []string `json:"has"` // credential keys present (never the values)
	CreatedAt string   `json:"created_at"`
}

func toDTO(a store.Account) accountDTO {
	has := make([]string, 0, len(a.Creds)+1)
	if a.Secret != "" {
		has = append(has, "secret")
	}
	for k := range a.Creds {
		has = append(has, k)
	}
	return accountDTO{
		ID:        a.ID,
		Provider:  a.Provider,
		Label:     a.Label,
		Status:    a.Status,
		Has:       has,
		CreatedAt: a.CreatedAt.Format("2006-01-02 15:04"),
	}
}

func (h *Accounts) List(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	rows, err := h.store.List(r.Context(), provider)
	if err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]accountDTO, 0, len(rows))
	for _, a := range rows {
		out = append(out, toDTO(a))
	}
	writeData(w, out)
}

type addAccountReq struct {
	Provider string            `json:"provider"`
	Label    string            `json:"label"`
	Secret   string            `json:"secret"`
	Creds    map[string]string `json:"creds"`
}

func (h *Accounts) Add(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var in addAccountReq
	if err := json.Unmarshal(body, &in); err != nil || in.Provider == "" {
		writeAPIErr(w, http.StatusBadRequest, "provider is required")
		return
	}
	if in.Secret == "" && len(in.Creds) == 0 {
		writeAPIErr(w, http.StatusBadRequest, "a secret or credentials are required")
		return
	}
	id, err := h.store.Add(r.Context(), store.Account{
		Provider: in.Provider,
		Label:    in.Label,
		Secret:   in.Secret,
		Creds:    in.Creds,
		Status:   "active",
	})
	if err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, map[string]any{"id": id})
}

type setStatusReq struct {
	Status string `json:"status"`
}

func (h *Accounts) SetStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeAPIErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in setStatusReq
	body, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(body, &in); err != nil || in.Status == "" {
		writeAPIErr(w, http.StatusBadRequest, "status is required")
		return
	}
	if err := h.store.SetStatus(r.Context(), id, in.Status); err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, map[string]any{"ok": true})
}

func (h *Accounts) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeAPIErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, map[string]any{"ok": true})
}
