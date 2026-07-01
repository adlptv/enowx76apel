package kiro

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/enowdev/enowx/core/provider"
)

// Usage reports the account's CodeWhisperer credit limits. Implements
// provider.UsageReporter.
func (p *Provider) Usage(acc provider.Account) (*provider.Usage, error) {
	am := p.manager(acc)
	token, err := am.token()
	if err != nil {
		return nil, err
	}
	region := orDefault(acc.Cred("sso_region"), "us-east-1")
	profileARN := am.profileARN()
	return fetchKiroUsage(p.doer, token, profileARN, region)
}

func fetchKiroUsage(doer interface {
	Do(*http.Request) (*http.Response, error)
}, token, profileARN, region string) (*provider.Usage, error) {
	values := url.Values{
		"origin":          {"AI_EDITOR"},
		"resourceType":    {"AGENTIC_REQUEST"},
		"isEmailRequired": {"true"},
	}
	if profileARN != "" {
		values.Set("profileArn", profileARN)
	}

	// Endpoints tried in order; the first that returns 200 wins.
	urls := []string{
		fmt.Sprintf("https://management.%s.kiro.dev/getUsageLimits?%s", region, values.Encode()),
		fmt.Sprintf("https://codewhisperer.%s.amazonaws.com/getUsageLimits?%s", region, values.Encode()),
		fmt.Sprintf("https://q.%s.amazonaws.com/getUsageLimits?%s", region, values.Encode()),
	}

	var lastErr error
	for _, u := range urls {
		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("x-amz-user-agent", "aws-sdk-js/1.0.0 KiroIDE")
		resp, err := doer.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("kiro usage %d", resp.StatusCode)
			continue
		}
		return parseKiroUsage(body)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("kiro usage unavailable")
	}
	return nil, lastErr
}

// Email fetches the account's email from getUsageLimits (isEmailRequired=true),
// so accounts added by token can still be labelled by email. "" if unavailable.
func (p *Provider) Email(acc provider.Account) string {
	am := p.manager(acc)
	token, err := am.token()
	if err != nil {
		return ""
	}
	region := orDefault(acc.Cred("sso_region"), "us-east-1")
	values := url.Values{
		"origin":          {"AI_EDITOR"},
		"resourceType":    {"AGENTIC_REQUEST"},
		"isEmailRequired": {"true"},
	}
	if arn := am.profileARN(); arn != "" {
		values.Set("profileArn", arn)
	}
	urls := []string{
		fmt.Sprintf("https://management.%s.kiro.dev/getUsageLimits?%s", region, values.Encode()),
		fmt.Sprintf("https://codewhisperer.%s.amazonaws.com/getUsageLimits?%s", region, values.Encode()),
		fmt.Sprintf("https://q.%s.amazonaws.com/getUsageLimits?%s", region, values.Encode()),
	}
	for _, u := range urls {
		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			continue
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("x-amz-user-agent", "aws-sdk-js/1.0.0 KiroIDE")
		resp, err := p.doer.Do(req)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			continue
		}
		var payload struct {
			UserInfo struct {
				Email string `json:"email"`
			} `json:"userInfo"`
		}
		if json.Unmarshal(body, &payload) == nil && payload.UserInfo.Email != "" {
			return payload.UserInfo.Email
		}
	}
	return ""
}

func parseKiroUsage(body []byte) (*provider.Usage, error) {
	var payload struct {
		PlanType           string `json:"planType"`
		UsageBreakdownList []struct {
			UsageLimit   float64 `json:"usageLimit"`
			CurrentUsage float64 `json:"currentUsage"`
		} `json:"usageBreakdownList"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	var limit, used float64
	for _, b := range payload.UsageBreakdownList {
		limit += b.UsageLimit
		used += b.CurrentUsage
	}
	u := &provider.Usage{Limit: limit, Used: used, Remaining: limit - used, Plan: payload.PlanType}
	if limit == 0 {
		u.Message = "no quota data"
	}
	return u, nil
}
