package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/enowdev/enowx/store"
)

// AutoClaw handles provider-specific endpoints for autoclaw accounts:
// wallet balance overview, batch login status, etc.
type AutoClaw struct {
	store store.AccountStore
}

func NewAutoClaw(s store.AccountStore) *AutoClaw {
	return &AutoClaw{store: s}
}

// GET /api/accounts/autoclaw/wallets returns all cached wallet balances.
func (h *AutoClaw) Wallets(w http.ResponseWriter, r *http.Request) {
	wallets := map[string]float64{}
	accs, err := h.store.List(r.Context(), "autoclaw")
	if err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, a := range accs {
		if a.Creds != nil {
			if v, ok := a.Creds["wallet_balance"]; ok {
				var b float64
				json.Unmarshal([]byte(v), &b)
				wallets[a.Creds["email"]] = b
			}
		}
	}
	writeData(w, map[string]any{"wallets": wallets, "total": len(accs)})
}

// POST /api/accounts/autoclaw/manual  { "json": "...", "label": "" }
func (h *AutoClaw) Manual(w http.ResponseWriter, r *http.Request) {
	var in struct {
		JSON  string `json:"json"`
		Label string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeAPIErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var raw map[string]string
	if err := json.Unmarshal([]byte(in.JSON), &raw); err != nil {
		writeAPIErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	creds := make(map[string]string)
	for k, v := range raw {
		creds[k] = v
	}
	if creds["access_token"] == "" && creds["accessToken"] != "" {
		creds["access_token"] = creds["accessToken"]
	}
	if creds["refresh_token"] == "" && creds["refreshToken"] != "" {
		creds["refresh_token"] = creds["refreshToken"]
	}

	if creds["access_token"] == "" {
		writeAPIErr(w, http.StatusBadRequest, "missing access_token in credentials")
		return
	}

	label := in.Label
	if label == "" {
		label = creds["email"]
	}
	if label == "" {
		label = creds["user_id"]
	}

	id, err := h.store.Add(r.Context(), store.Account{
		Provider: "autoclaw",
		Label:    label,
		Creds:    creds,
		Status:   "active",
	})
	if err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, map[string]any{"id": id})
}

// POST /api/accounts/autoclaw/refresh  { "refresh_token": "...", "source_id": "...", "device_id": "...", "label": "" }
func (h *AutoClaw) Refresh(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RefreshToken string `json:"refresh_token"`
		SourceID     string `json:"source_id"`
		DeviceID     string `json:"device_id"`
		Label        string `json:"label"`
	}
	json.NewDecoder(r.Body).Decode(&in)
	if in.RefreshToken == "" {
		writeAPIErr(w, http.StatusBadRequest, "refresh_token is required")
		return
	}
	creds := map[string]string{
		"refresh_token": in.RefreshToken,
		"source_id":     nz(in.SourceID, "autoclaw"),
		"device_id":     in.DeviceID,
	}
	id, err := h.store.Add(r.Context(), store.Account{
		Provider: "autoclaw",
		Label:    nz(in.Label, "autoclaw-"+in.SourceID),
		Creds:    creds,
		Status:   "active",
	})
	if err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, map[string]any{"id": id})
}
