// Package autoclaw is an OpenAI-compatible upstream for AutoClaw / Z.ai
// (AutoGLM) accounts. It signs requests with app-level MD5 headers and
// uses multi-field credentials (access_token, refresh_token, device_id).
package autoclaw

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/enowdev/enowx/core/convert"
	"github.com/enowdev/enowx/core/model"
	"github.com/enowdev/enowx/core/provider"
	"github.com/enowdev/enowx/core/provider/oaistream"
	"github.com/enowdev/enowx/core/transport"
)

// Provider is an OpenAI-on-the-wire provider for AutoClaw/Z.ai accounts.
// Each account stores access_token, refresh_token, device_id in Creds.
type Provider struct {
	doer transport.Doer
	save CredSaver

	mu       sync.Mutex
	managers map[int64]*authManager

	bg *Background // background wallet check + token refresher
}

func New(doer transport.Doer, save CredSaver) *Provider {
	return &Provider{doer: doer, save: save, managers: map[int64]*authManager{}}
}

func (p *Provider) Name() string        { return "autoclaw" }
func (p *Provider) Caps() provider.Caps { return provider.Caps{Chat: true} }

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

// SetBackground attaches the background manager and starts it.
func (p *Provider) SetBackground(bg *Background) {
	p.bg = bg
	if bg != nil {
		go bg.Start()
	}
}

// Background returns the background manager (nil if not started).
func (p *Provider) Background() *Background {
	return p.bg
}

// BuildRequest forwards the normalized request as OpenAI to the AutoClaw proxy.
func (p *Provider) BuildRequest(req *model.Request, acc provider.Account) (*http.Request, error) {
	am := p.manager(acc)
	token, err := am.token()
	if err != nil {
		return nil, fmt.Errorf("autoclaw: %w", err)
	}

	// Map client model -> upstream model header
	upstreamModel := defaultModel
	if m, ok := modelMap[req.Model]; ok {
		upstreamModel = m
	}

	// Use the raw body when the request came in as OpenAI, otherwise re-encode.
	body := convert.OpenAIBody(req)

	r, err := http.NewRequest(http.MethodPost, chatCompletion, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	r.Header = signedHeaders()
	r.Header.Set("X-Authorization", "Bearer "+token)
	r.Header.Set("X-Request-Model", upstreamModel)
	r.Header.Set("X-Trace-Id", fmt.Sprintf("enx-%d", time.Now().UnixNano()))

	return r, nil
}

// ParseResponse uses the standard OpenAI SSE/JSON parser.
func (p *Provider) ParseResponse(resp *http.Response, req *model.Request) (model.Stream, error) {
	return oaistream.Parse(resp, req.Stream)
}

// ── Outcome classification ──

func (p *Provider) Classify(status int, body []byte) provider.Outcome {
	switch {
	case status < 400:
		return provider.OutcomeOK
	case status == 401 || status == 403:
		return provider.OutcomeDead
	case status == 429 || bytes.Contains(body, []byte("insufficient_quota")):
		return provider.OutcomeExhausted
	case status >= 500:
		return provider.OutcomeTransient
	default:
		return provider.OutcomeOK
	}
}

// ── Wallet / Usage reporter ──

type walletResp struct {
	Code int `json:"code"`
	Data *struct {
		TotalBalance int `json:"total_balance"`
	} `json:"data"`
}

// Usage reports the wallet balance for an account (AutoClaw reward points).
func (p *Provider) Usage(acc provider.Account) (*provider.Usage, error) {
	am := p.manager(acc)
	token, err := am.token()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, walletURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header = signedHeaders()
	req.Header.Set("authorization", "Bearer "+token) // lowercase for assetmgr!

	resp, err := p.doer.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var wr walletResp
	if err := json.Unmarshal(raw, &wr); err != nil {
		return nil, err
	}
	if wr.Code != 0 || wr.Data == nil {
		return nil, fmt.Errorf("autoclaw wallet: code=%d", wr.Code)
	}

	remaining := float64(wr.Data.TotalBalance)
	return &provider.Usage{
		Limit:     max(remaining, 1), // avoid 0 limit UI quirks
		Used:      0,
		Remaining: remaining,
		Plan:      "autoclaw",
	}, nil
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// ── Model fetcher ──

func (p *Provider) Models(acc provider.Account) ([]provider.Model, error) {
	models := make([]provider.Model, 0, len(modelMap))
	for alias, upstream := range modelMap {
		models = append(models, provider.Model{
			ID:      alias,
			Name:    alias,
			Type:    "chat",
			OwnedBy: upstream,
		})
	}
	return models, nil
}

// Email resolves the account's email from its credentials.
func (p *Provider) Email(acc provider.Account) string {
	if e, ok := acc.Creds["email"]; ok && e != "" {
		return e
	}
	return ""
}

// init registers the autoclaw provider with the global registry when the
// package is imported. main.go also does an explicit Register call.
var registerOnce sync.Once

func init() {
	// We cannot access the global registry from here; main.go will register.
	// This init just ensures the package compiles. The actual registration
	// happens in cmd/enowx/main.go.
	log.SetPrefix("[autoclaw] ")
}
