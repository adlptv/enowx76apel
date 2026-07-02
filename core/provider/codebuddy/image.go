package codebuddy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/enowdev/enowx/core/provider"
	"github.com/enowdev/enowx/core/proxy"
	"github.com/enowdev/enowx/core/transport"
)

// GenerateImage runs a text-to-image request against codebuddy's image endpoint.
func (p *Provider) GenerateImage(doer transport.Doer, acc provider.Account, req provider.ImageRequest) (*provider.ImageResult, error) {
	model := req.Model
	if _, bare := proxy.SplitModel(model); bare != "" {
		model = bare
	}
	size := req.Size
	if size == "" {
		size = "1024x1024"
	}
	rf := req.ResponseFormat
	if rf == "" {
		rf = "b64_json"
	}
	quality := req.Quality
	if quality == "" {
		quality = "standard"
	}
	n := req.N
	if n <= 0 {
		n = 1
	}

	body, _ := json.Marshal(map[string]any{
		"model":           model,
		"prompt":          req.Prompt,
		"size":            size,
		"response_format": rf,
		"quality":         quality,
		"n":               n,
	})
	r, err := http.NewRequest(http.MethodPost, p.v.base+"/v2/images/generations", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	p.ids.apply(r.Header, p.v.domain, "Bearer "+strings.TrimSpace(acc.Cred("api_key")))
	r.Header.Set("Content-Type", "application/json")

	resp, err := doer.Do(r)
	if err != nil {
		return nil, fmt.Errorf("codebuddy image: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("codebuddy image %d: %s", resp.StatusCode, trunc(raw))
	}

	var out struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Data []struct {
				URL     string `json:"url"`
				B64JSON string `json:"b64_json"`
			} `json:"data"`
			Usage struct {
				Credit float64 `json:"credit"`
			} `json:"usage"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("codebuddy image decode: %w", err)
	}
	if out.Code != 0 {
		return nil, fmt.Errorf("codebuddy image error (%d): %s", out.Code, out.Msg)
	}
	res := &provider.ImageResult{Credit: out.Data.Usage.Credit}
	for _, d := range out.Data.Data {
		res.Images = append(res.Images, provider.ImageData{URL: d.URL, B64JSON: d.B64JSON})
	}
	return res, nil
}

func trunc(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 200 {
		return s[:200]
	}
	return s
}
