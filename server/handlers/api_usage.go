package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/enowdev/enowx/core/provider"
	"github.com/enowdev/enowx/store"
)

// Usage reports an account's credit/quota when its provider supports it.
type Usage struct {
	reg   *provider.Registry
	store store.AccountStore
}

func NewUsage(reg *provider.Registry, s store.AccountStore) *Usage {
	return &Usage{reg: reg, store: s}
}

func (h *Usage) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeAPIErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	acc, err := h.account(r, id)
	if err != nil {
		writeAPIErr(w, http.StatusNotFound, "account not found")
		return
	}
	prov, err := h.reg.Get(acc.Provider)
	if err != nil {
		writeAPIErr(w, http.StatusBadRequest, err.Error())
		return
	}
	reporter, ok := prov.(provider.UsageReporter)
	if !ok {
		writeData(w, map[string]any{"supported": false})
		return
	}
	usage, err := reporter.Usage(provider.Account{ID: acc.ID, Secret: acc.Secret, Creds: acc.Creds})
	if err != nil {
		writeAPIErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeData(w, map[string]any{"supported": true, "usage": usage})
}

// account finds an account by id across all providers.
func (h *Usage) account(r *http.Request, id int64) (store.Account, error) {
	rows, err := h.store.List(r.Context(), "")
	if err != nil {
		return store.Account{}, err
	}
	for _, a := range rows {
		if a.ID == id {
			return a, nil
		}
	}
	return store.Account{}, errNotFound
}

var errNotFound = &notFoundError{}

type notFoundError struct{}

func (*notFoundError) Error() string { return "not found" }
