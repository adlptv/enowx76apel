// Package compat is a single, definition-driven provider for user-added
// OpenAI- or Anthropic-compatible upstreams. One instance per custom provider.
package compat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/enowdev/enowx/core/convert"
	"github.com/enowdev/enowx/core/model"
	"github.com/enowdev/enowx/core/provider"
	"github.com/enowdev/enowx/core/provider/oaistream"
)

// Def is a custom-provider definition (from the store).
type Def struct {
	Name    string
	Format  string // "openai" | "anthropic"
	BaseURL string
	Models  []provider.Model
}

type Provider struct{ def Def }

func New(def Def) *Provider {
	def.BaseURL = strings.TrimRight(def.BaseURL, "/")
	if def.Format != "anthropic" {
		def.Format = "openai"
	}
	return &Provider{def: def}
}

func (p *Provider) Name() string        { return p.def.Name }
func (p *Provider) Caps() provider.Caps { return provider.Caps{Chat: true} }

func (p *Provider) BuildRequest(req *model.Request, acc provider.Account) (*http.Request, error) {
	key := strings.TrimSpace(acc.Cred("api_key"))
	if p.def.Format == "anthropic" {
		r, err := http.NewRequest(http.MethodPost, p.def.BaseURL+"/v1/messages", bytes.NewReader(anthropicBody(req)))
		if err != nil {
			return nil, err
		}
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("x-api-key", key)
		r.Header.Set("anthropic-version", "2023-06-01")
		return r, nil
	}
	r, err := http.NewRequest(http.MethodPost, p.def.BaseURL+"/chat/completions", bytes.NewReader(convert.OpenAIBody(req)))
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+key)
	return r, nil
}

func (p *Provider) ParseResponse(resp *http.Response, req *model.Request) (model.Stream, error) {
	if p.def.Format == "anthropic" {
		return parseAnthropic(resp, req.Stream)
	}
	return oaistream.Parse(resp, req.Stream)
}

func (p *Provider) Classify(status int, body []byte) provider.Outcome {
	switch {
	case status < 400:
		return provider.OutcomeOK
	case status == 401 || status == 403:
		return provider.OutcomeDead
	case status == 429 || bytes.Contains(body, []byte("insufficient")) || bytes.Contains(body, []byte("quota")):
		return provider.OutcomeExhausted
	case status >= 500:
		return provider.OutcomeTransient
	default:
		return provider.OutcomeOK
	}
}

// Models tries the upstream's /v1/models then /models (OpenAI shape); on failure
// falls back to the definition's manually-entered models.
func (p *Provider) Models(acc provider.Account) ([]provider.Model, error) {
	if live := p.fetchModels(acc); len(live) > 0 {
		return live, nil
	}
	if len(p.def.Models) > 0 {
		return p.def.Models, nil
	}
	return nil, fmt.Errorf("no models available")
}

func (p *Provider) fetchModels(acc provider.Account) []provider.Model {
	key := strings.TrimSpace(acc.Cred("api_key"))
	for _, path := range []string{"/v1/models", "/models"} {
		r, err := http.NewRequest(http.MethodGet, p.def.BaseURL+path, nil)
		if err != nil {
			continue
		}
		if p.def.Format == "anthropic" {
			r.Header.Set("x-api-key", key)
			r.Header.Set("anthropic-version", "2023-06-01")
		} else {
			r.Header.Set("Authorization", "Bearer "+key)
		}
		resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(r)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if resp.StatusCode >= 300 {
			continue
		}
		var out struct {
			Data []struct {
				ID      string `json:"id"`
				OwnedBy string `json:"owned_by"`
			} `json:"data"`
		}
		if json.Unmarshal(body, &out) != nil || len(out.Data) == 0 {
			continue
		}
		models := make([]provider.Model, 0, len(out.Data))
		for _, m := range out.Data {
			models = append(models, provider.Model{ID: m.ID, Name: m.ID, Type: "chat", OwnedBy: m.OwnedBy})
		}
		return models
	}
	return nil
}
