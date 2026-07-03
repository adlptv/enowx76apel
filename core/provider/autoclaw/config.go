// Package autoclaw is an OpenAI-compatible provider for AutoClaw / Z.ai
// (AutoGLM) accounts. It signs requests with app-level MD5 headers and uses
// multi-field credentials (access_token, refresh_token, device_id, user_id).
package autoclaw

import "net/http"

// ── App Signing ──
const (
	appID    = "100003"
	appKey   = "38d2391985e2369a5fb8227d8e6cd5e5"
	product  = "autoclaw"
	version  = "1.9.1"
	platform = "win"
)

// ── Endpoints ──
const (
	userAPIBase    = "https://autoglm-api.autoglm.ai"
	llmProxyBase   = "https://autoglm-api.autoglm.ai/autoclaw-proxy/proxy/autoclaw"
	chatCompletion = llmProxyBase + "/chat/completions"
	refreshURL     = userAPIBase + "/userapi/v1/refresh"
	profileURL     = userAPIBase + "/userapi/v1/user-profile"
	walletURL      = userAPIBase + "/agent-assetmgr/api/v2/wallets?biz_app_id=autoclaw"
	ledgerURL      = userAPIBase + "/agent-assetmgr/api/v1/ledgers_std?asset_type=point&wallet_type=all"
)

// ── Token TTL ──
const (
	accessTokenTTL = 86400 // 24 hours
	refreshMargin  = 300   // 5 min before expiry — trigger refresh
)

// ── Model Map ──
// modelMap keys are what the client sends as "model" in the OpenAI body;
// values are the X-Request-Model header sent to the AutoClaw upstream.
var modelMap = map[string]string{
	// Best — real GLM-5.2
	"glm-5.2":     "openrouter_glm-5.2",
	"glm-5.2-true": "openrouter_glm-5.2",
	// Cheapest — glm-5-turbo
	"glm-5-turbo": "zai_glm-5-turbo",
	"cheap":       "zai_glm-5-turbo",
	// Avoid — secretly DeepSeek-V4-Pro ~7x cost
	"auto":     "zai_auto",
	"deepseek": "zai_auto",
}

var defaultModel = "openrouter_glm-5.2"

// signHeaders builds the app-level signing headers that the AutoClaw API requires.
func signHeaders() http.Header {
	h := make(http.Header)
	h.Set("Content-Type", "application/json")
	h.Set("X-Auth-Appid", appID)
	h.Set("X-Product", product)
	h.Set("X-Version", version)
	h.Set("X-Tm", platform)
	// X-Trace-Id is set per-request in BuildRequest so each call has a unique one.
	// TimeStamp and Sign are set in BuildRequest too because they need fresh values.
	return h
}
