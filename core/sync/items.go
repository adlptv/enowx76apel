package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/enowdev/enowx/store"
)

func nowMillis() int64 { return time.Now().UnixMilli() }

// Sync item types beyond playlists. Must match the cloud's gated set.
const (
	typeCustomProvider = "custom_provider"
	typeAccount        = "account"
	typeAPIKey         = "apikey"
	typeAlias          = "alias"
)

func shortHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])[:16]
}

// --- entitlement + key from cached /me ---

type cachedMe struct {
	SyncKey      string   `json:"sync_key"`
	KDFSalt      string   `json:"kdf_salt"`
	Entitlements []string `json:"entitlements"`
}

func (m *Manager) cachedMe(ctx context.Context) cachedMe {
	var me cachedMe
	_ = json.Unmarshal([]byte(m.get(ctx, keyUser)), &me)
	return me
}

// hasFullSync reports whether the logged-in user may sync the gated types.
func (m *Manager) hasFullSync(ctx context.Context) bool {
	for _, e := range m.cachedMe(ctx).Entitlements {
		if e == "cloud.sync.full" {
			return true
		}
	}
	return false
}

// credKey derives the AES key for sealing credentials, or nil if unavailable.
func (m *Manager) credKey(ctx context.Context) []byte {
	me := m.cachedMe(ctx)
	key, err := deriveKey(me.SyncKey, me.KDFSalt)
	if err != nil {
		return nil
	}
	return key
}

// --- snapshot: local rows → sync items ---

// fullSyncItems adds accounts/keys/aliases/custom-providers to the snapshot when
// the user is entitled. Plaintext for definitions/aliases, AES-GCM for creds.
func (m *Manager) fullSyncItems(ctx context.Context, out map[string]item) {
	if !m.hasFullSync(ctx) {
		return
	}
	key := m.credKey(ctx)

	// Custom providers (plaintext) — keyed by prefix (stable across devices).
	if m.custom != nil {
		if list, err := m.custom.List(ctx); err == nil {
			for _, cp := range list {
				payload, _ := json.Marshal(cp)
				id := typeCustomProvider + ":" + cp.Prefix
				out[id] = item{ItemID: id, Type: typeCustomProvider, Version: 1, UpdatedAt: nowMillis(), Payload: string(payload)}
			}
		}
	}

	// Accounts (encrypted) — keyed by provider + a hash of the credentials, so
	// the same account maps to the same id on every device.
	if m.accounts != nil && key != nil {
		if list, err := m.accounts.List(ctx, ""); err == nil {
			for _, a := range list {
				raw, _ := json.Marshal(syncAccount{Provider: a.Provider, Label: a.Label, Secret: a.Secret, Creds: a.Creds, Status: a.Status, Disabled: a.Disabled})
				payload, nonce, err := seal(key, raw)
				if err != nil {
					continue
				}
				id := typeAccount + ":" + a.Provider + ":" + shortHash(a.Secret+fmt.Sprint(a.Creds))
				out[id] = item{ItemID: id, Type: typeAccount, Version: 1, UpdatedAt: nowMillis(), Encrypted: true, Payload: payload, Nonce: nonce}
			}
		}
	}

	// Gateway API keys (encrypted) — keyed by a hash of the secret.
	if m.keys != nil && key != nil {
		if list, err := m.keys.List(ctx); err == nil {
			for _, k := range list {
				raw, _ := json.Marshal(syncKey{Label: k.Label, Secret: k.Secret, TokenLimit: k.TokenLimit, MaxConcurrent: k.MaxConcurrent, Enabled: k.Enabled})
				payload, nonce, err := seal(key, raw)
				if err != nil {
					continue
				}
				id := typeAPIKey + ":" + shortHash(k.Secret)
				out[id] = item{ItemID: id, Type: typeAPIKey, Version: 1, UpdatedAt: nowMillis(), Encrypted: true, Payload: payload, Nonce: nonce}
			}
		}
	}

	// Model aliases (plaintext) — keyed by alias.
	if m.aliases != nil {
		if list, err := m.aliases.List(ctx); err == nil {
			for _, al := range list {
				payload, _ := json.Marshal(al)
				id := typeAlias + ":" + al.Alias
				out[id] = item{ItemID: id, Type: typeAlias, Version: 1, UpdatedAt: nowMillis(), Payload: string(payload)}
			}
		}
	}
}

// syncAccount / syncKey are the on-the-wire (encrypted) shapes.
type syncAccount struct {
	Provider string            `json:"provider"`
	Label    string            `json:"label"`
	Secret   string            `json:"secret"`
	Creds    map[string]string `json:"creds"`
	Status   string            `json:"status"`
	Disabled bool              `json:"disabled"`
}

type syncKey struct {
	Label         string `json:"label"`
	Secret        string `json:"secret"`
	TokenLimit    int64  `json:"token_limit"`
	MaxConcurrent int64  `json:"max_concurrent"`
	Enabled       bool   `json:"enabled"`
}

// --- apply: pulled items → local rows ---

// applyFullItem applies one non-playlist pulled item. Returns true if handled.
func (m *Manager) applyFullItem(ctx context.Context, ri item) bool {
	switch ri.Type {
	case typeCustomProvider:
		return m.applyCustomProvider(ctx, ri)
	case typeAccount:
		return m.applyAccount(ctx, ri)
	case typeAPIKey:
		return m.applyAPIKey(ctx, ri)
	case typeAlias:
		return m.applyAlias(ctx, ri)
	}
	return false
}

func (m *Manager) applyCustomProvider(ctx context.Context, ri item) bool {
	if m.custom == nil {
		return false
	}
	var cp store.CustomProvider
	if json.Unmarshal([]byte(ri.Payload), &cp) != nil {
		return false
	}
	// Upsert by prefix: skip if we already have this prefix, else create + register.
	existing, _ := m.custom.List(ctx)
	for _, e := range existing {
		if e.Prefix == cp.Prefix {
			return true // already present (LWW: keep local)
		}
	}
	if ri.Deleted {
		return true
	}
	id, err := m.custom.Create(ctx, cp)
	if err != nil {
		return false
	}
	cp.ID = id
	if m.onCustomProvider != nil {
		m.onCustomProvider(cp) // register live
	}
	return true
}

func (m *Manager) applyAccount(ctx context.Context, ri item) bool {
	if m.accounts == nil || !ri.Encrypted {
		return false
	}
	key := m.credKey(ctx)
	if key == nil {
		return false
	}
	raw, err := open(key, ri.Payload, ri.Nonce)
	if err != nil {
		return false
	}
	var sa syncAccount
	if json.Unmarshal(raw, &sa) != nil {
		return false
	}
	// Dedup: skip if an account with the same secret+creds already exists.
	existing, _ := m.accounts.List(ctx, sa.Provider)
	target := shortHash(sa.Secret + fmt.Sprint(sa.Creds))
	for _, e := range existing {
		if shortHash(e.Secret+fmt.Sprint(e.Creds)) == target {
			return true
		}
	}
	if ri.Deleted {
		return true
	}
	_, _ = m.accounts.Add(ctx, store.Account{Provider: sa.Provider, Label: sa.Label, Secret: sa.Secret, Creds: sa.Creds, Status: sa.Status, Disabled: sa.Disabled})
	return true
}

func (m *Manager) applyAPIKey(ctx context.Context, ri item) bool {
	if m.keys == nil || !ri.Encrypted {
		return false
	}
	key := m.credKey(ctx)
	if key == nil {
		return false
	}
	raw, err := open(key, ri.Payload, ri.Nonce)
	if err != nil {
		return false
	}
	var sk syncKey
	if json.Unmarshal(raw, &sk) != nil {
		return false
	}
	if existing, _ := m.keys.BySecret(ctx, sk.Secret); existing != nil {
		return true // already have it
	}
	if ri.Deleted {
		return true
	}
	_, _ = m.keys.Add(ctx, store.APIKey{Label: sk.Label, Secret: sk.Secret, TokenLimit: sk.TokenLimit, MaxConcurrent: sk.MaxConcurrent, Enabled: sk.Enabled})
	return true
}

func (m *Manager) applyAlias(ctx context.Context, ri item) bool {
	if m.aliases == nil {
		return false
	}
	var al store.ModelAlias
	if json.Unmarshal([]byte(ri.Payload), &al) != nil {
		return false
	}
	if ri.Deleted {
		_ = m.aliases.Delete(ctx, al.Alias)
		return true
	}
	_ = m.aliases.Set(ctx, al.Alias, al.Target)
	return true
}
