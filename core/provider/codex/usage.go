package codex

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/enowdev/enowx/core/provider"
)

const usageURL = "https://chatgpt.com/backend-api/wham/usage"

// Usage reports the account's plan + rate-limit headroom. Codex has no numeric
// credit, so we surface the primary rate-limit window as used/limit percentages
// and the plan as the label.
func (p *Provider) Usage(acc provider.Account) (*provider.Usage, error) {
	am := p.manager(acc)
	token, err := am.token()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, usageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	resp, err := p.doer.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return &provider.Usage{Message: "usage unavailable"}, nil
	}

	type window struct {
		UsedPercent        float64 `json:"used_percent"`
		LimitWindowSeconds int64   `json:"limit_window_seconds"`
		ResetAfterSeconds  int64   `json:"reset_after_seconds"`
	}
	var out struct {
		PlanType  string `json:"plan_type"`
		RateLimit struct {
			Primary   *window `json:"primary_window"`
			Secondary *window `json:"secondary_window"`
		} `json:"rate_limit"`
	}
	if json.Unmarshal(body, &out) != nil {
		return &provider.Usage{Message: "usage parse failed"}, nil
	}

	windows := []provider.UsageWindow{}
	add := func(w *window) {
		if w == nil {
			return
		}
		windows = append(windows, provider.UsageWindow{
			Label:       windowLabel(w.LimitWindowSeconds),
			UsedPercent: w.UsedPercent,
			ResetInSecs: w.ResetAfterSeconds,
		})
	}
	add(out.RateLimit.Primary)
	add(out.RateLimit.Secondary)

	u := &provider.Usage{Plan: out.PlanType, Windows: windows}
	// Keep the legacy single meter populated from the primary window so anything
	// not window-aware still shows something.
	if out.RateLimit.Primary != nil {
		u.Limit = 100
		u.Used = out.RateLimit.Primary.UsedPercent
		u.Remaining = 100 - u.Used
	} else {
		u.Message = out.PlanType
	}
	return u, nil
}

// windowLabel names a rate-limit window from its length.
func windowLabel(seconds int64) string {
	switch {
	case seconds <= 0:
		return "window"
	case seconds <= 6*3600:
		return "5h"
	case seconds >= 6*86400:
		return "Weekly"
	default:
		return "Daily"
	}
}
