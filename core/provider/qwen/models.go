package qwen

import "github.com/enowdev/enowx/core/provider"

// catalog is the fallback model list when the live /v1/models fetch fails.
func catalog() []provider.Model {
	return []provider.Model{
		{ID: "qwen3-coder-plus", Name: "Qwen3 Coder Plus", Type: "chat", OwnedBy: "qwen"},
		{ID: "qwen3-coder-flash", Name: "Qwen3 Coder Flash", Type: "chat", OwnedBy: "qwen"},
		{ID: "coder-model", Name: "Qwen3.6 Coder Model", Type: "chat", OwnedBy: "qwen"},
		{ID: "vision-model", Name: "Qwen3 Vision Model", Type: "image", OwnedBy: "qwen"},
	}
}

// modelType classifies a fetched model id (vision models render as image).
func modelType(id string) string {
	if id == "vision-model" {
		return "image"
	}
	return "chat"
}
