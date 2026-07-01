package antigravity

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ParseManualJSON accepts a pasted Antigravity credentials JSON (snake or
// camelCase) and returns canonical creds.
func ParseManualJSON(s string) (map[string]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty JSON")
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		return nil, err
	}
	get := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := raw[k].(string); ok && v != "" {
				return v
			}
		}
		return ""
	}
	access := get("access_token", "accessToken")
	if access == "" {
		return nil, fmt.Errorf("missing access_token")
	}
	creds := map[string]string{
		"access_token":  access,
		"refresh_token": get("refresh_token", "refreshToken"),
	}
	if p := get("project_id", "projectId"); p != "" {
		creds["project_id"] = p
	}
	if e := get("email"); e != "" {
		creds["email"] = e
	}
	if exp := get("expires_at", "expiresAt"); exp != "" {
		if _, err := time.Parse(time.RFC3339, exp); err == nil {
			creds["expires_at"] = exp
		}
	}
	return creds, nil
}
