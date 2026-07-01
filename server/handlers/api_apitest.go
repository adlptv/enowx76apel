package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/enowdev/enowx/store"
)

// ApiTest is the persistence API for the Postman-style dev tool: collections,
// saved requests, environments, and run history. All local (not synced).
type ApiTest struct{ store store.ApiTestStore }

func NewApiTest(s store.ApiTestStore) *ApiTest { return &ApiTest{store: s} }

func idParam(r *http.Request) int64 {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	return id
}

// GET /api/apitest — everything the UI needs in one shot.
func (h *ApiTest) All(w http.ResponseWriter, r *http.Request) {
	cols, err := h.store.Collections(r.Context())
	if err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	reqs, _ := h.store.Requests(r.Context())
	envs, _ := h.store.Environments(r.Context())
	hist, _ := h.store.History(r.Context(), 100)
	writeData(w, map[string]any{"collections": cols, "requests": reqs, "environments": envs, "history": hist})
}

// --- collections ---

func (h *ApiTest) AddCollection(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	readJSON(r, &in)
	if in.Name == "" {
		in.Name = "New collection"
	}
	id, err := h.store.AddCollection(r.Context(), in.Name)
	if err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, map[string]any{"id": id})
}

func (h *ApiTest) RenameCollection(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	readJSON(r, &in)
	if err := h.store.RenameCollection(r.Context(), idParam(r), in.Name); err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, map[string]any{"ok": true})
}

func (h *ApiTest) DeleteCollection(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeleteCollection(r.Context(), idParam(r)); err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, map[string]any{"ok": true})
}

// --- requests ---

func (h *ApiTest) SaveRequest(w http.ResponseWriter, r *http.Request) {
	var req store.ApiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIErr(w, http.StatusBadRequest, "bad body")
		return
	}
	id, err := h.store.SaveRequest(r.Context(), req)
	if err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, map[string]any{"id": id})
}

func (h *ApiTest) DeleteRequest(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeleteRequest(r.Context(), idParam(r)); err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, map[string]any{"ok": true})
}

// --- environments ---

func (h *ApiTest) SaveEnvironment(w http.ResponseWriter, r *http.Request) {
	var env store.ApiEnvironment
	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		writeAPIErr(w, http.StatusBadRequest, "bad body")
		return
	}
	id, err := h.store.SaveEnvironment(r.Context(), env)
	if err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, map[string]any{"id": id})
}

func (h *ApiTest) DeleteEnvironment(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeleteEnvironment(r.Context(), idParam(r)); err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, map[string]any{"ok": true})
}

func (h *ApiTest) ActivateEnvironment(w http.ResponseWriter, r *http.Request) {
	if err := h.store.SetActiveEnvironment(r.Context(), idParam(r)); err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, map[string]any{"ok": true})
}

// --- history ---

func (h *ApiTest) AddHistory(w http.ResponseWriter, r *http.Request) {
	var hh store.ApiHistory
	if err := json.NewDecoder(r.Body).Decode(&hh); err != nil {
		writeAPIErr(w, http.StatusBadRequest, "bad body")
		return
	}
	_ = h.store.AddHistory(r.Context(), hh)
	writeData(w, map[string]any{"ok": true})
}

func (h *ApiTest) ClearHistory(w http.ResponseWriter, r *http.Request) {
	if err := h.store.ClearHistory(r.Context()); err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, map[string]any{"ok": true})
}
