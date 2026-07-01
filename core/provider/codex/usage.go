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

	var out struct {
		Email     string `json:"email"`
		PlanType  string `json:"plan_type"`
		Summary   struct {
			Plan string `json:"plan"`
		} `json:"summary"`
		RateLimit struct {
			Primary struct {
				UsedPercent        float64 `json:"used_percent"`
				LimitWindowSeconds int64   `json:"limit_window_seconds"`
				ResetAfterSeconds  int64   `json:"reset_after_seconds"`
			} `json:"primary_window"`
		} `json:"rate_limit"`
	}
	if json.Unmarshal(body, &out) != nil {
		return &provider.Usage{Message: "usage parse failed"}, nil
	}

	plan := out.PlanType
	if plan == "" {
		plan = out.Summary.Plan
	}
	used := out.RateLimit.Primary.UsedPercent
	u := &provider.Usage{
		Limit:     100,
		Used:      used,
		Remaining: 100 - used,
		Plan:      plan,
	}
	if out.RateLimit.Primary.LimitWindowSeconds == 0 {
		u.Message = plan
	}
	return u, nil
}
