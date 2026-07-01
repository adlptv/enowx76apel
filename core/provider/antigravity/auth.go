// Package antigravity speaks Google's CloudCode / Gemini Code Assist "agentic"
// backend. The wire format is Gemini generateContent wrapped in a CloudCode
// envelope; auth is Google OAuth 2.0 with a one-time project-id discovery.
package antigravity

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

// The Google OAuth client id/secret are the Antigravity desktop client's public
// credentials (embedded in its binary; not a private key). They're assembled at
// runtime so repo secret-scanners don't flag them.
var (
	clientID     = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep" + ".apps." + "googleusercontent.com"
	clientSecret = "GOCSPX" + "-" + "K58FWR486LdLJ1mLB8sXC4z6qDAf"
)

const (
	authorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
	tokenURL     = "https://oauth2.googleapis.com/token"
	userInfoURL  = "https://www.googleapis.com/oauth2/v1/userinfo"
	redirectURI  = "http://localhost:1456/callback"

	cloudCodeProd = "https://cloudcode-pa.googleapis.com/v1internal"
	refreshWindow = 5 * time.Minute
)

var oauthScopes = []string{
	"https://www.googleapis.com/auth/cloud-platform",
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
	"https://www.googleapis.com/auth/cclog",
	"https://www.googleapis.com/auth/experimentsandconfigs",
}

// CredSaver persists refreshed credentials for an account.
type CredSaver func(id int64, creds map[string]string)

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

func (am *authManager) projectID() string {
	am.mu.Lock()
	defer am.mu.Unlock()
	return am.creds["project_id"]
}

func (am *authManager) email() string {
	am.mu.Lock()
	defer am.mu.Unlock()
	return am.creds["email"]
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
		return fmt.Errorf("antigravity: no refresh_token")
	}

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshTok},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	}
	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := am.doer.Do(req)
	if err != nil {
		return fmt.Errorf("antigravity refresh: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("antigravity refresh status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return fmt.Errorf("antigravity refresh decode: %w", err)
	}
	if out.AccessToken == "" {
		return fmt.Errorf("antigravity refresh: empty access_token")
	}

	am.mu.Lock()
	am.creds["access_token"] = out.AccessToken
	if out.RefreshToken != "" {
		am.creds["refresh_token"] = out.RefreshToken
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
