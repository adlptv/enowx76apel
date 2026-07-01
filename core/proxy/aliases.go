package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"time"
)

// AliasFetcher provides the current alias→model_id map (from the cloud).
type AliasFetcher interface {
	ModelAliases(ctx context.Context) map[string]string
}

// AliasResolver caches the cloud alias→model_id map and rewrites request models
// (and the raw body's "model" field) so a user can call a model by an alias.
type AliasResolver struct {
	fetch AliasFetcher
	ttl   time.Duration

	mu      sync.RWMutex
	aliases map[string]string
	fetched time.Time
}

func NewAliasResolver(fetch AliasFetcher, ttl time.Duration) *AliasResolver {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &AliasResolver{fetch: fetch, ttl: ttl, aliases: map[string]string{}}
}

// Resolve returns the real model id for a possibly-aliased model. The cached map
// is refreshed lazily when stale. Unknown models pass through unchanged.
func (r *AliasResolver) Resolve(ctx context.Context, model string) string {
	r.mu.RLock()
	stale := time.Since(r.fetched) > r.ttl
	real, ok := r.aliases[model]
	r.mu.RUnlock()

	if stale {
		if m := r.fetch.ModelAliases(ctx); m != nil {
			r.mu.Lock()
			r.aliases = m
			r.fetched = time.Now()
			real, ok = m[model]
			r.mu.Unlock()
		}
	}
	if ok && real != "" {
		return real
	}
	return model
}

// RewriteBody replaces the top-level "model" field in an OpenAI/Anthropic JSON
// body with real, if it differs. Returns the (possibly unchanged) body.
func RewriteBody(raw json.RawMessage, alias, real string) json.RawMessage {
	if alias == real || len(raw) == 0 {
		return raw
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw
	}
	b, _ := json.Marshal(real)
	m["model"] = b
	out, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	// Preserve compactness; json.Marshal already produces compact output.
	return bytes.TrimSpace(out)
}
