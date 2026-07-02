package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/enowdev/enowx/core/plugins"
	syncpkg "github.com/enowdev/enowx/core/sync"
	"github.com/enowdev/enowx/server/middleware"
)

// Market handles publishing/browsing/installing plugins via the cloud, and the
// admin scan settings. Dashboard-gated (loopback).
type Market struct {
	dash *middleware.Dashboard
	sync *syncpkg.Manager
	mgr  *plugins.Manager
}

func NewMarket(dash *middleware.Dashboard, sm *syncpkg.Manager, mgr *plugins.Manager) *Market {
	return &Market{dash: dash, sync: sm, mgr: mgr}
}

func (h *Market) guard(w http.ResponseWriter, r *http.Request) bool {
	if !h.dash.Authorized(r) {
		writeAPIErr(w, http.StatusForbidden, "requires the dashboard login when accessed remotely")
		return false
	}
	return true
}

func (h *Market) rawJSON(w http.ResponseWriter, raw string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(raw))
}

// POST /api/market/publish {id} — bundle the local plugin + publish for scanning.
func (h *Market) Publish(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w, r) {
		return
	}
	var in struct{ ID string }
	body, _ := io.ReadAll(r.Body)
	_ = json.Unmarshal(body, &in)
	man, err := h.mgr.Get(in.ID)
	if err != nil {
		writeAPIErr(w, http.StatusBadRequest, err.Error())
		return
	}
	zipBytes, err := h.mgr.Bundle(in.ID)
	if err != nil {
		writeAPIErr(w, http.StatusBadRequest, err.Error())
		return
	}
	fields := map[string]string{
		"name": man.Name, "slug": man.ID, "description": man.Description,
		"runtime": man.Runtime, "version": "1.0.0",
	}
	raw, err := h.sync.PublishPlugin(r.Context(), fields, zipBytes)
	if err != nil {
		writeAPIErr(w, http.StatusBadGateway, err.Error())
		return
	}
	h.rawJSON(w, raw)
}

// GET /api/market/plugins?q= — browse published plugins.
func (h *Market) List(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w, r) {
		return
	}
	q := ""
	if v := strings.TrimSpace(r.URL.Query().Get("q")); v != "" {
		q = "?q=" + v
	}
	raw, err := h.sync.MarketPlugins(r.Context(), q)
	if err != nil {
		writeAPIErr(w, http.StatusBadGateway, err.Error())
		return
	}
	h.rawJSON(w, raw)
}

// POST /api/market/install/{id} — download the bundle + extract into plugins/.
func (h *Market) Install(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w, r) {
		return
	}
	id := chi.URLParam(r, "id")
	raw, err := h.sync.InstallPlugin(r.Context(), id)
	if err != nil {
		writeAPIErr(w, http.StatusBadGateway, err.Error())
		return
	}
	var meta struct {
		Data struct {
			BundleURL string `json:"bundle_url"`
			Slug      string `json:"slug"`
		} `json:"data"`
	}
	_ = json.Unmarshal([]byte(raw), &meta)
	if meta.Data.BundleURL == "" || meta.Data.Slug == "" {
		writeAPIErr(w, http.StatusBadGateway, "missing bundle info")
		return
	}
	zipBytes, err := h.sync.DownloadBundle(r.Context(), meta.Data.BundleURL)
	if err != nil {
		writeAPIErr(w, http.StatusBadGateway, err.Error())
		return
	}
	if err := h.mgr.Extract(meta.Data.Slug, zipBytes); err != nil {
		writeAPIErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeData(w, map[string]any{"installed": true, "id": meta.Data.Slug})
}

// GET /api/admin/plugin-scan — the AI scan settings (proxied to cloud).
func (h *Market) GetScanSettings(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w, r) {
		return
	}
	raw, err := h.sync.AdminSettings(r.Context())
	if err != nil {
		writeAPIErr(w, http.StatusBadGateway, err.Error())
		return
	}
	h.rawJSON(w, raw)
}

// PUT /api/admin/plugin-scan — update the AI scan settings.
func (h *Market) SaveScanSettings(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w, r) {
		return
	}
	body, _ := io.ReadAll(r.Body)
	raw, err := h.sync.SaveAdminSettings(r.Context(), body)
	if err != nil {
		writeAPIErr(w, http.StatusBadGateway, err.Error())
		return
	}
	h.rawJSON(w, raw)
}
