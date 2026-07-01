package antigravity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/enowdev/enowx/core/model"
	"github.com/enowdev/enowx/core/provider"
	"github.com/enowdev/enowx/core/transport"
)

const (
	inferenceHost = "https://daily-cloudcode-pa.googleapis.com/v1internal"
	userAgentBase = "antigravity/1.107.0"
)

type Provider struct {
	doer transport.Doer
	save CredSaver

	mu       sync.Mutex
	managers map[int64]*authManager
}

func New(doer transport.Doer, save CredSaver) *Provider {
	return &Provider{doer: doer, save: save, managers: map[int64]*authManager{}}
}

func (p *Provider) Name() string        { return "antigravity" }
func (p *Provider) Caps() provider.Caps { return provider.Caps{Chat: true, Images: true} }

func (p *Provider) manager(acc provider.Account) *authManager {
	p.mu.Lock()
	defer p.mu.Unlock()
	if am, ok := p.managers[acc.ID]; ok {
		return am
	}
	am := newAuthManager(p.doer, p.save, acc)
	p.managers[acc.ID] = am
	return am
}

func (p *Provider) BuildRequest(req *model.Request, acc provider.Account) (*http.Request, error) {
	am := p.manager(acc)
	token, err := am.token()
	if err != nil {
		return nil, err
	}
	session := sessionID(am.email(), acc.ID)
	body, reverse := buildEnvelope(req, am.projectID(), session)

	url := inferenceHost + ":generateContent"
	if req.Stream {
		url = inferenceHost + ":streamGenerateContent?alt=sse"
	}
	r, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("User-Agent", userAgentBase+" "+goPlatform())
	r.Header.Set("X-Machine-Session-Id", session)
	if req.Stream {
		r.Header.Set("Accept", "text/event-stream")
	} else {
		r.Header.Set("Accept", "application/json")
	}
	if len(reverse) > 0 {
		r = r.WithContext(withReverseNames(r.Context(), reverse))
	}
	return r, nil
}

func (p *Provider) ParseResponse(resp *http.Response, req *model.Request) (model.Stream, error) {
	return newAntigravityStream(resp, reverseNamesFrom(resp.Request.Context())), nil
}

func (p *Provider) Classify(status int, _ []byte) provider.Outcome {
	switch {
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		return provider.OutcomeDead
	case status == http.StatusTooManyRequests:
		return provider.OutcomeExhausted
	default:
		return provider.OutcomeTransient
	}
}

func (p *Provider) Models(_ provider.Account) ([]provider.Model, error) {
	return catalog(), nil
}

func (p *Provider) Email(acc provider.Account) string {
	if e := acc.Cred("email"); e != "" {
		return e
	}
	return ""
}

// --- context plumbing for the reverse tool-name map ---

type reverseKey struct{}

func withReverseNames(ctx context.Context, m map[string]string) context.Context {
	return context.WithValue(ctx, reverseKey{}, m)
}

func reverseNamesFrom(ctx context.Context) map[string]string {
	if ctx == nil {
		return nil
	}
	if m, ok := ctx.Value(reverseKey{}).(map[string]string); ok {
		return m
	}
	return nil
}

func goPlatform() string {
	// e.g. "darwin/arm64"
	return runtime.GOOS + "/" + runtime.GOARCH
}

// sessionID derives a stable-per-account session id (used for prompt caching and
// echoed in the X-Machine-Session-Id header).
func sessionID(email string, accID int64) string {
	seed := email
	if seed == "" {
		seed = itoa(int(accID))
	}
	h := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(h[:8]) + itoa(int(time.Now().UnixMilli()%1_000_000))
}
