package qwen

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/enowdev/enowx/core/provider"
	"github.com/enowdev/enowx/core/transport"
)

const (
	clientID      = "f0304373b74a44d2b584a3fb70ca9e56"
	tokenURL      = "https://chat.qwen.ai/api/v1/oauth2/token"
	deviceCodeURL = "https://chat.qwen.ai/api/v1/oauth2/device/code"
	oauthScope    = "openid profile email model.completion"
	defaultHost   = "portal.qwen.ai"
	refreshWindow = 5 * time.Minute
)

// CredSaver persists refreshed credentials for an account.
type CredSaver func(id int64, creds map[string]string)

// authManager holds one account's tokens and refreshes them on demand.
type authManager struct {
	mu      sync.Mutex
	doer    transport.Doer
	save    CredSaver
	id      int64
	creds   map[string]string
	expires time.Time
}

func newAuthManager(doer transport.Doer, save CredSaver, acc provider.Account) *authManager {
	creds := make(map[string]string, len(acc.Creds))
	maps.Copy(creds, acc.Creds)
	am := &authManager{doer: doer, save: save, id: acc.ID, creds: creds}
	if exp := creds["expires_at"]; exp != "" {
		if t, err := time.Parse(time.RFC3339, exp); err == nil {
			am.expires = t
		}
	}
	return am
}

func (am *authManager) resourceURL() string {
	am.mu.Lock()
	defer am.mu.Unlock()
	return am.creds["resource_url"]
}

// token returns a valid access token, refreshing when expired or about to be.
func (am *authManager) token() (string, error) {
	am.mu.Lock()
	tok := am.creds["access_token"]
	soon := am.expires.IsZero() || time.Until(am.expires) < refreshWindow
	am.mu.Unlock()

	if tok != "" && !soon {
		return tok, nil
	}
	if err := am.refresh(); err != nil {
		if tok != "" {
			return tok, nil
		}
		return "", err
	}
	am.mu.Lock()
	defer am.mu.Unlock()
	return am.creds["access_token"], nil
}

func (am *authManager) refresh() error {
	am.mu.Lock()
	refreshTok := am.creds["refresh_token"]
	am.mu.Unlock()
	if refreshTok == "" {
		return fmt.Errorf("qwen: no refresh_token")
	}

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshTok},
		"client_id":     {clientID},
	}
	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	resp, err := am.doer.Do(req)
	if err != nil {
		return fmt.Errorf("qwen refresh: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qwen refresh status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		ResourceURL  string `json:"resource_url"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return fmt.Errorf("qwen refresh decode: %w", err)
	}
	if out.AccessToken == "" {
		return fmt.Errorf("qwen refresh: empty access_token")
	}

	am.mu.Lock()
	am.creds["access_token"] = out.AccessToken
	if out.RefreshToken != "" {
		am.creds["refresh_token"] = out.RefreshToken
	}
	if out.ResourceURL != "" {
		am.creds["resource_url"] = out.ResourceURL
	}
	if out.ExpiresIn > 0 {
		am.expires = time.Now().Add(time.Duration(out.ExpiresIn-60) * time.Second)
	} else {
		am.expires = time.Now().Add(time.Hour)
	}
	am.creds["expires_at"] = am.expires.Format(time.RFC3339)
	snapshot := make(map[string]string, len(am.creds))
	maps.Copy(snapshot, am.creds)
	am.mu.Unlock()

	if am.save != nil {
		am.save(am.id, snapshot)
	}
	return nil
}

// host extracts the bare host from a resource_url (scheme + trailing slash
// stripped), defaulting to portal.qwen.ai.
func host(resourceURL string) string {
	h := strings.TrimSpace(resourceURL)
	if h == "" {
		return defaultHost
	}
	h = strings.TrimPrefix(h, "https://")
	h = strings.TrimPrefix(h, "http://")
	h = strings.TrimRight(h, "/")
	if h == "" {
		return defaultHost
	}
	return h
}

// endpointFor returns the chat/completions endpoint for a resource_url.
func endpointFor(resourceURL string) string {
	return "https://" + host(resourceURL) + "/v1/chat/completions"
}

// modelsURLFor returns the /v1/models endpoint for a resource_url.
func modelsURLFor(resourceURL string) string {
	return "https://" + host(resourceURL) + "/v1/models"
}
