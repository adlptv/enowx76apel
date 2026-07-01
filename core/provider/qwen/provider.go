// Package qwen speaks Qwen Code (Alibaba's coding subscription). It is
// OpenAI-compatible on the wire, so it reuses the OpenAI body encoder and stream
// parser; only the OAuth token refresh and the per-account resource_url endpoint
// are provider-specific.
package qwen

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"github.com/enowdev/enowx/core/convert"
	"github.com/enowdev/enowx/core/model"
	"github.com/enowdev/enowx/core/provider"
	"github.com/enowdev/enowx/core/provider/oaistream"
	"github.com/enowdev/enowx/core/proxy"
	"github.com/enowdev/enowx/core/transport"
)

const userAgent = "QwenCode/0.12.3 (linux; x64)"

type Provider struct {
	doer transport.Doer
	save CredSaver

	mu       sync.Mutex
	managers map[int64]*authManager
}

func New(doer transport.Doer, save CredSaver) *Provider {
	return &Provider{doer: doer, save: save, managers: map[int64]*authManager{}}
}

func (p *Provider) Name() string        { return "qwen" }
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
	// Strip the qw/ prefix so upstream sees the bare model id (rewrite the raw
	// body's model field to match, then re-encode as OpenAI chat/completions).
	if _, bare := proxy.SplitModel(req.Model); bare != req.Model {
		req.Raw = proxy.RewriteBody(req.Raw, req.Model, bare)
		req.Model = bare
	}
	r, err := http.NewRequest(http.MethodPost, endpointFor(am.resourceURL()), bytes.NewReader(convert.OpenAIBody(req)))
	if err != nil {
		return nil, err
	}
	setQwenHeaders(r.Header, token, req.Stream)
	return r, nil
}

func (p *Provider) ParseResponse(resp *http.Response, req *model.Request) (model.Stream, error) {
	return oaistream.Parse(resp, req.Stream)
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

// Models fetches the account's models from the upstream /v1/models, falling back
// to the hardcoded catalog on failure.
func (p *Provider) Models(acc provider.Account) ([]provider.Model, error) {
	am := p.manager(acc)
	token, err := am.token()
	if err != nil {
		return catalog(), nil
	}
	r, err := http.NewRequest(http.MethodGet, modelsURLFor(am.resourceURL()), nil)
	if err != nil {
		return catalog(), nil
	}
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("User-Agent", userAgent)
	resp, err := p.doer.Do(r)
	if err != nil {
		return catalog(), nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return catalog(), nil
	}
	var out struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &out) != nil || len(out.Data) == 0 {
		return catalog(), nil
	}
	models := make([]provider.Model, 0, len(out.Data))
	for _, m := range out.Data {
		owner := m.OwnedBy
		if owner == "" {
			owner = "qwen"
		}
		models = append(models, provider.Model{ID: m.ID, Name: m.ID, Type: modelType(m.ID), OwnedBy: owner})
	}
	return models, nil
}

// setQwenHeaders applies the Qwen Code identity headers upstream expects.
func setQwenHeaders(h http.Header, token string, stream bool) {
	h.Set("Authorization", "Bearer "+token)
	h.Set("Content-Type", "application/json")
	h.Set("User-Agent", userAgent)
	h.Set("X-DashScope-AuthType", "qwen-oauth")
	h.Set("X-DashScope-UserAgent", userAgent)
	if stream {
		h.Set("Accept", "text/event-stream")
	} else {
		h.Set("Accept", "application/json")
	}
}
