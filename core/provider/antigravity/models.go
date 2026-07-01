package antigravity

import "github.com/enowdev/enowx/core/provider"

// catalog is the hardcoded Antigravity model list (no live /models endpoint).
func catalog() []provider.Model {
	return []provider.Model{
		{ID: "gemini-3.1-pro-low", Name: "Gemini 3.1 Pro (Low)", Type: "chat", OwnedBy: "google"},
		{ID: "gemini-pro-agent", Name: "Gemini 3.1 Pro (High)", Type: "chat", OwnedBy: "google"},
		{ID: "gemini-3.5-flash-low", Name: "Gemini 3.5 Flash (Medium)", Type: "chat", OwnedBy: "google"},
		{ID: "gemini-3-flash-agent", Name: "Gemini 3.5 Flash (High)", Type: "chat", OwnedBy: "google"},
		{ID: "gemini-3.5-flash-extra-low", Name: "Gemini 3.5 Flash (Extra Low)", Type: "chat", OwnedBy: "google"},
		{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6 (Thinking)", Type: "chat", OwnedBy: "anthropic"},
		{ID: "claude-opus-4-6-thinking", Name: "Claude Opus 4.6 (Thinking)", Type: "chat", OwnedBy: "anthropic"},
		{ID: "gpt-oss-120b-medium", Name: "GPT-OSS 120B (Medium)", Type: "chat", OwnedBy: "openai"},
	}
}
