package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/enowdev/enowx/core/provider/qwen"
	"github.com/enowdev/enowx/core/transport"
	"github.com/enowdev/enowx/store"
)

// Qwen handles the device-code add flow for Qwen accounts.
type Qwen struct {
	doer   transport.Doer
	store  store.AccountStore
	warmer Warmer

	mu       sync.Mutex
	sessions map[string]*qwenSession
	seq      int64
}

type qwenSession struct {
	deviceCode string
	verifier   string
	created    time.Time
}

func NewQwen(doer transport.Doer, s store.AccountStore) *Qwen {
	return &Qwen{doer: doer, store: s, sessions: map[string]*qwenSession{}}
}

// SetWarmer enables automatic warmup of newly-added qwen accounts.
func (h *Qwen) SetWarmer(w Warmer) { h.warmer = w }

func (h *Qwen) id() string {
	h.seq++
	return time.Now().Format("150405") + "-" + itoa(h.seq)
}

// POST /api/accounts/qwen/device/start -> {session, user_code, verification_uri, ...}
func (h *Qwen) DeviceStart(w http.ResponseWriter, r *http.Request) {
	dev, err := qwen.StartDevice(h.doer)
	if err != nil {
		writeAPIErr(w, http.StatusBadGateway, err.Error())
		return
	}
	h.mu.Lock()
	sid := h.id()
	h.sessions[sid] = &qwenSession{deviceCode: dev.DeviceCode, verifier: dev.CodeVerifier, created: time.Now()}
	h.mu.Unlock()
	writeData(w, map[string]any{
		"session":                   sid,
		"user_code":                 dev.UserCode,
		"verification_uri":          dev.VerificationURI,
		"verification_uri_complete": dev.VerificationURIComplete,
		"interval":                  dev.Interval,
	})
}

// GET /api/accounts/qwen/device/poll?session= -> {status: pending|done, id?}
func (h *Qwen) DevicePoll(w http.ResponseWriter, r *http.Request) {
	sid := r.URL.Query().Get("session")
	h.mu.Lock()
	s := h.sessions[sid]
	h.mu.Unlock()
	if s == nil {
		writeAPIErr(w, http.StatusNotFound, "unknown session")
		return
	}
	creds, pending, err := qwen.PollDevice(h.doer, s.deviceCode, s.verifier)
	if err != nil {
		writeAPIErr(w, http.StatusBadGateway, err.Error())
		return
	}
	if pending {
		writeData(w, map[string]any{"status": "pending"})
		return
	}
	h.mu.Lock()
	delete(h.sessions, sid)
	h.mu.Unlock()

	id, err := h.store.Add(r.Context(), store.Account{Provider: "qwen", Creds: creds, Status: "active"})
	if err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := map[string]any{"status": "done", "id": id}
	if warm := autoWarm(r.Context(), h.warmer, h.store, id); warm != nil {
		out["warmup"] = warm
	}
	writeData(w, out)
}
