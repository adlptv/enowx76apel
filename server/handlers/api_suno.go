package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/enowdev/enowx/core/provider"
	"github.com/enowdev/enowx/core/proxy"
	"github.com/enowdev/enowx/core/suno"
	"github.com/enowdev/enowx/store"
)

// Suno exposes AI music generation (create task + poll). Generation goes through
// the proxy so it rotates accounts + marks exhausted ones; polling reads the key
// straight from a pooled Suno account.
type Suno struct {
	store  store.AccountStore
	proxy  *proxy.Proxy
	client *suno.Client
}

func NewSuno(s store.AccountStore, p *proxy.Proxy, client *suno.Client) *Suno {
	return &Suno{store: s, proxy: p, client: client}
}

// keys returns the api_keys of all enabled Suno accounts (for polling).
func (h *Suno) keys(r *http.Request) []string {
	rows, err := h.store.List(r.Context(), "suno")
	if err != nil {
		return nil
	}
	out := []string{}
	for _, a := range rows {
		if a.Disabled {
			continue
		}
		if k := strings.TrimSpace(a.Creds["api_key"]); k != "" {
			out = append(out, k)
		}
	}
	return out
}

// GET /api/music/suno/key -> { configured }
func (h *Suno) GetKey(w http.ResponseWriter, r *http.Request) {
	writeData(w, map[string]any{"configured": len(h.keys(r)) > 0})
}

// POST /api/music/generate { prompt, style?, title?, model?, instrumental?, custom_mode? }
func (h *Suno) Generate(w http.ResponseWriter, r *http.Request) {
	if len(h.keys(r)) == 0 {
		writeAPIErr(w, http.StatusBadRequest, "no Suno account configured (add one in Providers)")
		return
	}
	var in struct {
		Prompt       string `json:"prompt"`
		Style        string `json:"style"`
		Title        string `json:"title"`
		Model        string `json:"model"`
		Instrumental bool   `json:"instrumental"`
		CustomMode   bool   `json:"custom_mode"`
	}
	body, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(body, &in); err != nil {
		writeAPIErr(w, http.StatusBadRequest, "invalid request")
		return
	}
	if strings.TrimSpace(in.Prompt) == "" && !in.CustomMode {
		writeAPIErr(w, http.StatusBadRequest, "prompt is required")
		return
	}
	// Through the proxy: rotates to another Suno account + marks exhausted ones
	// when credits run out.
	res, err := h.proxy.GenerateMusic(r.Context(), "suno", provider.MusicRequest{
		Prompt: in.Prompt, Style: in.Style, Title: in.Title, Model: in.Model,
		Instrumental: in.Instrumental, CustomMode: in.CustomMode,
	})
	if err != nil {
		writeAPIErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeData(w, map[string]any{"task_id": res.TaskID})
}

// GET /api/music/generate/status?task_id=...
func (h *Suno) Status(w http.ResponseWriter, r *http.Request) {
	keys := h.keys(r)
	if len(keys) == 0 {
		writeAPIErr(w, http.StatusBadRequest, "no Suno account configured (add one in Providers)")
		return
	}
	taskID := strings.TrimSpace(r.URL.Query().Get("task_id"))
	if taskID == "" {
		writeAPIErr(w, http.StatusBadRequest, "task_id is required")
		return
	}
	// The task belongs to whichever account created it — try each key until one
	// resolves the task (has tracks or a terminal status).
	var res *suno.TaskResult
	var lastErr error
	for _, key := range keys {
		res, lastErr = h.client.Poll(key, taskID)
		if lastErr == nil && res != nil && (len(res.Tracks) > 0 || res.Done || res.Failed) {
			break
		}
	}
	if res == nil {
		writeAPIErr(w, http.StatusBadGateway, "poll failed")
		return
	}
	writeData(w, res)
}
