package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/enowdev/enowx/core/provider"
	"github.com/enowdev/enowx/store"
)

// Images serves the OpenAI-style image generation endpoint at
// /v1/images/generations.
func (h *V1) Images(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "read body")
		return
	}
	var in struct {
		Model          string `json:"model"`
		Prompt         string `json:"prompt"`
		N              int    `json:"n"`
		Size           string `json:"size"`
		Quality        string `json:"quality"`
		ResponseFormat string `json:"response_format"`
	}
	if err := json.Unmarshal(body, &in); err != nil || in.Model == "" {
		writeErr(w, http.StatusBadRequest, "invalid request")
		return
	}
	if in.Prompt == "" {
		writeErr(w, http.StatusBadRequest, "prompt is required")
		return
	}

	// Resolve alias, then route by the (possibly prefixed) model id.
	modelID := in.Model
	if h.resolver != nil {
		modelID = h.resolver.Resolve(r.Context(), modelID)
	}
	providerName := h.route(modelID)

	res, err := h.proxy.GenerateImage(r.Context(), providerName, provider.ImageRequest{
		Model:          modelID,
		Prompt:         in.Prompt,
		N:              in.N,
		Size:           in.Size,
		Quality:        in.Quality,
		ResponseFormat: in.ResponseFormat,
	})
	if err != nil {
		h.imgLog(providerName, in.Model, "error", start)
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}

	data := make([]map[string]any, 0, len(res.Images))
	for _, img := range res.Images {
		d := map[string]any{}
		if img.URL != "" {
			d["url"] = img.URL
		}
		if img.B64JSON != "" {
			d["b64_json"] = img.B64JSON
		}
		data = append(data, d)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"created": time.Now().Unix(), "data": data})
	h.imgLog(providerName, in.Model, "success", start)
}

func (h *V1) imgLog(providerName, modelID, status string, start time.Time) {
	if h.logs == nil {
		return
	}
	_ = h.logs.Insert(context.Background(), store.RequestLog{
		Provider:  providerName,
		Model:     modelID,
		Status:    status,
		Source:    "image",
		LatencyMS: time.Since(start).Milliseconds(),
	})
}
