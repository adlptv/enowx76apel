package handlers

import (
	"encoding/json"
	"net/http"

	syncpkg "github.com/enowdev/enowx/core/sync"
)

// Sync exposes the cloud-sync controls to the local UI. The actual protocol
// talks to the enowxlabs server; this just drives it.
type Sync struct{ mgr *syncpkg.Manager }

func NewSync(mgr *syncpkg.Manager) *Sync { return &Sync{mgr: mgr} }

func (h *Sync) Status(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var user any
	if u := h.mgr.UserJSON(ctx); u != "" {
		_ = json.Unmarshal([]byte(u), &user)
	}
	writeData(w, map[string]any{
		"configured": h.mgr.Configured(ctx),
		"enabled":    h.mgr.Enabled(ctx),
		"server_url": h.mgr.ServerURL(ctx),
		"user":       user,
	})
}

// LoginStart returns the Discord authorize URL + state to poll.
func (h *Sync) LoginStart(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ServerURL string `json:"server_url"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	url, state, err := h.mgr.LoginStart(r.Context(), body.ServerURL)
	if err != nil {
		writeAPIErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeData(w, map[string]any{"authorize_url": url, "state": state})
}

// LoginPoll checks whether the browser flow completed.
func (h *Sync) LoginPoll(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	done, userJSON, err := h.mgr.LoginPoll(r.Context(), state)
	if err != nil {
		writeAPIErr(w, http.StatusBadGateway, err.Error())
		return
	}
	var user any
	if userJSON != "" {
		_ = json.Unmarshal([]byte(userJSON), &user)
	}
	writeData(w, map[string]any{"done": done, "user": user})
}

func (h *Sync) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.mgr.Logout(r.Context()); err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, map[string]any{"ok": true})
}

// Now runs a one-off reconcile.
func (h *Sync) Now(w http.ResponseWriter, r *http.Request) {
	pushed, pulled, err := h.mgr.Sync(r.Context())
	if err != nil {
		writeAPIErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeData(w, map[string]any{"pushed": pushed, "pulled": pulled})
}
