// Package suno registers Suno as a pool provider. Suno does music generation
// (not chat), so it only advertises the Music capability; the actual generation
// runs through the /api/music/* endpoints using a pooled account's api_key.
package suno

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/enowdev/enowx/core/model"
	"github.com/enowdev/enowx/core/provider"
	"github.com/enowdev/enowx/core/suno"
	"github.com/enowdev/enowx/core/transport"
)

type Provider struct{ client *suno.Client }

func New(doer transport.Doer) *Provider { return &Provider{client: suno.New(doer)} }

func (p *Provider) Name() string        { return "suno" }
func (p *Provider) Caps() provider.Caps { return provider.Caps{Music: true} }

// GenerateMusic starts a Suno generation and returns the task id.
func (p *Provider) GenerateMusic(_ transport.Doer, acc provider.Account, req provider.MusicRequest) (*provider.MusicResult, error) {
	key := strings.TrimSpace(acc.Cred("api_key"))
	if key == "" {
		return nil, fmt.Errorf("suno account has no api_key")
	}
	taskID, err := p.client.Generate(key, suno.GenerateRequest{
		Prompt: req.Prompt, Model: req.Model, Style: req.Style, Title: req.Title,
		Instrumental: req.Instrumental, CustomMode: req.CustomMode,
	})
	if err != nil {
		return nil, err
	}
	return &provider.MusicResult{TaskID: taskID}, nil
}

// Usage reports the account's remaining Suno credits.
func (p *Provider) Usage(acc provider.Account) (*provider.Usage, error) {
	key := strings.TrimSpace(acc.Cred("api_key"))
	if key == "" {
		return &provider.Usage{Message: "no api_key"}, nil
	}
	credits, err := p.client.Credit(key)
	if err != nil {
		return &provider.Usage{Message: "credits unavailable"}, nil
	}
	// Suno bills per generation (~10 credits each). We don't get a max, so expose
	// remaining as the message; the bar uses remaining/limit where limit is a
	// nominal ceiling so a non-empty balance reads as healthy.
	return &provider.Usage{Remaining: credits, Limit: 100, Message: fmt.Sprintf("%.0f credits", credits)}, nil
}

// Suno is not a chat/completions upstream — it's driven by the music handlers.
func (p *Provider) BuildRequest(*model.Request, provider.Account) (*http.Request, error) {
	return nil, fmt.Errorf("suno does not support chat completions")
}

func (p *Provider) ParseResponse(*http.Response, *model.Request) (model.Stream, error) {
	return nil, fmt.Errorf("suno does not support chat completions")
}

func (p *Provider) Classify(status int, _ []byte) provider.Outcome {
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return provider.OutcomeDead
	}
	if status == http.StatusTooManyRequests {
		return provider.OutcomeExhausted
	}
	return provider.OutcomeTransient
}

// Models returns the Suno model versions available for generation. Suno has no
// live models endpoint, so this is a static catalog.
func (p *Provider) Models(provider.Account) ([]provider.Model, error) {
	return []provider.Model{
		{ID: "V5_5", Name: "Suno v5.5", Type: "music", OwnedBy: "suno"},
		{ID: "V5", Name: "Suno v5", Type: "music", OwnedBy: "suno"},
		{ID: "V4_5PLUS", Name: "Suno v4.5+", Type: "music", OwnedBy: "suno"},
		{ID: "V4_5", Name: "Suno v4.5", Type: "music", OwnedBy: "suno"},
		{ID: "V4", Name: "Suno v4", Type: "music", OwnedBy: "suno"},
	}, nil
}
