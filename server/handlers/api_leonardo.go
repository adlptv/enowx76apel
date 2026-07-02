package handlers

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/enowdev/enowx/core/leonardo"
	"github.com/enowdev/enowx/core/transport"
	"github.com/enowdev/enowx/store"
)

// Leonardo handles the Leonardo add flows: launch a real browser and read the
// session over CDP, exchange a pasted cookie, or accept a pasted token.
type Leonardo struct {
	store  store.AccountStore
	client *leonardo.Client
	warmer Warmer

	mu       sync.Mutex
	browsers map[string]*leonardo.Session
	seq      int64
}

func NewLeonardo(s store.AccountStore, doer transport.Doer) *Leonardo {
	return &Leonardo{store: s, client: leonardo.New(doer), browsers: map[string]*leonardo.Session{}}
}

func (h *Leonardo) SetWarmer(w Warmer) { h.warmer = w }

func (h *Leonardo) saveCreds(w http.ResponseWriter, r *http.Request, label string, c *leonardo.SessionCreds) {
	creds := map[string]string{"access_token": c.AccessToken}
	if c.CognitoSub != "" {
		creds["cognito_sub"] = c.CognitoSub
	}
	if c.Email != "" {
		creds["email"] = c.Email
	}
	id, err := h.store.Add(r.Context(), store.Account{
		Provider: "leonardo", Label: nz(label, c.Email), Creds: creds, Status: "active",
	})
	if err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := map[string]any{"ready": true, "id": id, "email": c.Email}
	if warm := autoWarm(r.Context(), h.warmer, h.store, id); warm != nil {
		out["warmup"] = warm
	}
	writeData(w, out)
}

// POST /api/accounts/leonardo/browser/start -> { session }
func (h *Leonardo) BrowserStart(w http.ResponseWriter, _ *http.Request) {
	sess, err := leonardo.LaunchChrome()
	if err != nil {
		writeAPIErr(w, http.StatusBadRequest, err.Error())
		return
	}
	h.mu.Lock()
	h.seq++
	id := time.Now().Format("150405") + "-" + itoa(h.seq)
	h.browsers[id] = sess
	h.mu.Unlock()
	// Auto-clean an abandoned browser after 10 minutes.
	time.AfterFunc(10*time.Minute, func() { h.closeBrowser(id) })
	writeData(w, map[string]any{"session": id})
}

// POST /api/accounts/leonardo/browser/poll { session }
func (h *Leonardo) BrowserPoll(w http.ResponseWriter, r *http.Request) {
	var in struct{ Session, Label string }
	readJSON(r, &in)
	h.mu.Lock()
	sess := h.browsers[in.Session]
	h.mu.Unlock()
	if sess == nil {
		writeAPIErr(w, http.StatusNotFound, "unknown session (browser closed?)")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	creds, err := sess.EvalGetSession(ctx)
	if err == leonardo.ErrNotReady {
		writeData(w, map[string]any{"ready": false})
		return
	}
	if err != nil {
		writeData(w, map[string]any{"ready": false, "note": err.Error()})
		return
	}
	h.closeBrowser(in.Session)
	h.saveCreds(w, r, in.Label, creds)
}

// POST /api/accounts/leonardo/browser/cancel { session }
func (h *Leonardo) BrowserCancel(w http.ResponseWriter, r *http.Request) {
	var in struct{ Session string }
	readJSON(r, &in)
	h.closeBrowser(in.Session)
	writeData(w, map[string]any{"ok": true})
}

func (h *Leonardo) closeBrowser(id string) {
	h.mu.Lock()
	sess := h.browsers[id]
	delete(h.browsers, id)
	h.mu.Unlock()
	if sess != nil {
		sess.Close()
	}
}

// POST /api/accounts/leonardo/cookie { cookie, label }
func (h *Leonardo) FromCookie(w http.ResponseWriter, r *http.Request) {
	var in struct{ Cookie, Label string }
	readJSON(r, &in)
	creds, err := h.client.SessionFromCookie(in.Cookie)
	if err != nil {
		writeAPIErr(w, http.StatusBadGateway, err.Error())
		return
	}
	c := map[string]string{"access_token": creds.AccessToken}
	if creds.CognitoSub != "" {
		c["cognito_sub"] = creds.CognitoSub
	}
	if creds.Email != "" {
		c["email"] = creds.Email
	}
	id, err := h.store.Add(r.Context(), store.Account{
		Provider: "leonardo",
		Label:    nz(in.Label, creds.Email),
		Creds:    c,
		Status:   "active",
	})
	if err != nil {
		writeAPIErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := map[string]any{"id": id, "email": creds.Email}
	if warm := autoWarm(r.Context(), h.warmer, h.store, id); warm != nil {
		out["warmup"] = warm
	}
	writeData(w, out)
}
