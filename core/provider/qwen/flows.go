package qwen

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/enowdev/enowx/core/transport"
)

// DeviceAuth is the device-code start result shown to the user.
type DeviceAuth struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	Interval                int    `json:"interval"`
	CodeVerifier            string `json:"-"`
}

// StartDevice begins the Qwen device-code PKCE flow.
func StartDevice(doer transport.Doer) (*DeviceAuth, error) {
	verifier, err := randB64URL(32)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	form := url.Values{
		"client_id":             {clientID},
		"scope":                 {oauthScope},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	req, err := http.NewRequest(http.MethodPost, deviceCodeURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	resp, err := doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("qwen device start: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qwen device start %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out DeviceAuth
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("qwen device decode: %w", err)
	}
	if out.DeviceCode == "" {
		return nil, fmt.Errorf("qwen device start: no device_code")
	}
	if out.Interval <= 0 {
		out.Interval = 5
	}
	out.CodeVerifier = verifier
	return &out, nil
}

// PollDevice polls the token endpoint once. pending=true means keep polling;
// creds!=nil means done.
func PollDevice(doer transport.Doer, deviceCode, verifier string) (creds map[string]string, pending bool, err error) {
	form := url.Values{
		"grant_type":    {"urn:ietf:params:oauth:grant-type:device_code"},
		"client_id":     {clientID},
		"device_code":   {deviceCode},
		"code_verifier": {verifier},
	}
	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	resp, err := doer.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("qwen poll: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		ResourceURL  string `json:"resource_url"`
		Error        string `json:"error"`
	}
	_ = json.Unmarshal(raw, &out)

	if out.AccessToken == "" {
		if out.Error == "authorization_pending" || out.Error == "slow_down" {
			return nil, true, nil
		}
		return nil, false, fmt.Errorf("qwen poll: %s", firstNonEmpty(out.Error, strings.TrimSpace(string(raw))))
	}

	exp := time.Now().Add(time.Hour)
	if out.ExpiresIn > 0 {
		exp = time.Now().Add(time.Duration(out.ExpiresIn-60) * time.Second)
	}
	c := map[string]string{
		"access_token":  out.AccessToken,
		"refresh_token": out.RefreshToken,
		"expires_at":    exp.Format(time.RFC3339),
	}
	if out.ResourceURL != "" {
		c["resource_url"] = out.ResourceURL
	}
	return c, false, nil
}

func randB64URL(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
