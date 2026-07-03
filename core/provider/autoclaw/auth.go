package autoclaw

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/enowdev/enowx/core/provider"
)

// ── Signing helpers ──

func freshTimestamp() string {
	return fmt.Sprintf("%d", time.Now().Unix())
}

func md5Sign(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// signedHeaders returns a Header set with fresh app-level signing headers.
func signedHeaders() http.Header {
	ts := freshTimestamp()
	h := make(http.Header)
	h.Set("Content-Type", "application/json")
	h.Set("X-Auth-Appid", appID)
	h.Set("X-Auth-TimeStamp", ts)
	h.Set("X-Auth-Sign", md5Sign(appID+"&"+ts+"&"+appKey))
	h.Set("X-Product", product)
	h.Set("X-Version", version)
	h.Set("X-Tm", platform)
	return h
}

// ── Token management per account ──

// CredSaver persists refreshed credentials for an account.
type CredSaver func(id int64, creds map[string]string)

// authManager holds one account's tokens and refreshes them on demand.
type authManager struct {
	mu      sync.Mutex
	doer    httpDoer
	save    CredSaver
	id      int64
	creds   map[string]string
	expires time.Time
}

// httpDoer is the subset of transport.Doer we need (avoids import cycle).
type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

func newAuthManager(doer httpDoer, save CredSaver, acc provider.Account) *authManager {
	creds := make(map[string]string, len(acc.Creds))
	for k, v := range acc.Creds {
		creds[k] = v
	}
	am := &authManager{doer: doer, save: save, id: acc.ID, creds: creds}
	if exp := creds["expires_at"]; exp != "" {
		t, err := time.Parse(time.RFC3339, exp)
		if err == nil {
			am.expires = t
		}
	}
	return am
}

// token returns a valid access token, refreshing first if needed.
func (am *authManager) token() (string, error) {
	am.mu.Lock()
	tok := am.creds["access_token"]
	needsRefresh := !am.expires.IsZero() && time.Now().Add(refreshMargin*time.Second).After(am.expires)
	am.mu.Unlock()

	if tok != "" && !needsRefresh {
		return tok, nil
	}
	if err := am.refresh(); err != nil {
		if tok != "" {
			return tok, nil // fall back to existing token
		}
		return "", err
	}
	am.mu.Lock()
	defer am.mu.Unlock()
	return am.creds["access_token"], nil
}

// refresh calls the AutoClaw refresh endpoint.
func (am *authManager) refresh() error {
	am.mu.Lock()
	refreshTok := am.creds["refresh_token"]
	sourceID := am.creds["source_id"]
	deviceID := am.creds["device_id"]
	am.mu.Unlock()

	if sourceID == "" {
		sourceID = "autoclaw"
	}
	if refreshTok == "" {
		return fmt.Errorf("autoclaw: no refresh_token for account %d", am.id)
	}

	body, _ := json.Marshal(map[string]string{
		"source_id":     sourceID,
		"device_id":     deviceID,
		"refresh_token": refreshTok,
	})

	req, err := http.NewRequest(http.MethodPost, refreshURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("autoclaw refresh: %w", err)
	}
	req.Header = signedHeaders()

	resp, err := am.doer.Do(req)
	if err != nil {
		return fmt.Errorf("autoclaw refresh: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var result struct {
		Code int `json:"code"`
		Data *struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return fmt.Errorf("autoclaw refresh decode: %w", err)
	}
	if result.Code != 0 || result.Data == nil || result.Data.AccessToken == "" {
		return fmt.Errorf("autoclaw refresh failed: code=%d body=%s", result.Code, string(raw))
	}

	am.mu.Lock()
	am.creds["access_token"] = result.Data.AccessToken
	if result.Data.RefreshToken != "" {
		am.creds["refresh_token"] = result.Data.RefreshToken
	}
	newExpiry := time.Now().Add(accessTokenTTL * time.Second)
	am.expires = newExpiry
	am.creds["expires_at"] = newExpiry.Format(time.RFC3339)

	snapshot := make(map[string]string, len(am.creds))
	for k, v := range am.creds {
		snapshot[k] = v
	}
	am.mu.Unlock()

	if am.save != nil {
		am.save(am.id, snapshot)
	}
	log.Printf("[autoclaw] Refreshed token for account %d", am.id)
	return nil
}
