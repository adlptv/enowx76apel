package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/enowdev/enowx/core/provider/antigravity"
	"github.com/enowdev/enowx/core/transport"
	"github.com/enowdev/enowx/store"
)

// Antigravity handles the OAuth + manual add flows for Antigravity accounts.
type Antigravity struct {
	doer   transport.Doer
	store  store.AccountStore
	warmer Warmer

	mu    sync.Mutex
	oauth map[string]*antigravitySession
	seq   int64
}

type antigravitySession struct {
	state   string
	created time.Time
}

func NewAntigravity(doer transport.Doer, s store.AccountStore) *Antigravity {
	return &Antigravity{doer: doer, store: s, oauth: map[string]*antigravitySession{}}
}

// SetWarmer enables automatic warmup of newly-added antigravity accounts.
func (h *Antigravity) SetWarmer(w Warmer) { h.warmer = w }

func (h *Antigravity) id() string {
	h.seq++
	return time.Now().Format("150405") + "-" + itoa(h.seq)
}

func (h *Antigravity) save(w http.ResponseWriter, r *http.Request, label string, creds map[string]string) {
	if creds["access_token"] == "" {
		writeAPIErr(w, http.StatusBadRequest, "missing access token in credentials")
		return
	}
	id, err := h.store.Add(r.Context(), store.Account{
		Provider: "antigravity",
		Label:    nz(label, creds["email"]),
		Creds:    creds,
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

// POST /api/accounts/antigravity/oauth/start -> {session, authorize_url}
func (h *Antigravity) OAuthStart(w http.ResponseWriter, _ *http.Request) {
	state, authURL, err := antigravity.AuthorizeURL()
	if err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.mu.Lock()
	sid := h.id()
	h.oauth[sid] = &antigravitySession{state: state, created: time.Now()}
	h.mu.Unlock()
	writeData(w, map[string]any{"session": sid, "authorize_url": authURL})
}

// POST /api/accounts/antigravity/oauth/exchange { session, code }
// code may be a raw code OR the full callback URL.
func (h *Antigravity) OAuthExchange(w http.ResponseWriter, r *http.Request) {
	var in struct{ Session, Code string }
	readJSON(r, &in)
	h.mu.Lock()
	s := h.oauth[in.Session]
	h.mu.Unlock()
	if s == nil {
		writeAPIErr(w, http.StatusNotFound, "unknown session")
		return
	}
	code := extractCode(in.Code)
	if code == "" {
		writeAPIErr(w, http.StatusBadRequest, "no auth code found in the pasted value")
		return
	}
	creds, err := antigravity.ExchangeAndOnboard(h.doer, code)
	if err != nil {
		writeAPIErr(w, http.StatusBadGateway, err.Error())
		return
	}
	h.mu.Lock()
	delete(h.oauth, in.Session)
	h.mu.Unlock()
	h.save(w, r, "", creds)
}

// POST /api/accounts/antigravity/manual { json, label }
func (h *Antigravity) Manual(w http.ResponseWriter, r *http.Request) {
	var in struct{ JSON, Label string }
	readJSON(r, &in)
	creds, err := antigravity.ParseManualJSON(in.JSON)
	if err != nil {
		writeAPIErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	h.save(w, r, in.Label, creds)
}
