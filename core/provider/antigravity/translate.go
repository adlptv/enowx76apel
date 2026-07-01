package antigravity

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/enowdev/enowx/core/model"
	"github.com/enowdev/enowx/core/proxy"
)

const maxOutputTokens = 16384

// buildEnvelope translates a normalized request into the CloudCode/Antigravity
// envelope (Gemini generateContent wrapped with agent metadata). Returns the
// JSON body and a sanitized→original tool-name map for restoring names in the
// response stream.
func buildEnvelope(req *model.Request, projectID, sessionID string) ([]byte, map[string]string) {
	upstream := req.Model
	if _, bare := proxy.SplitModel(req.Model); bare != "" {
		upstream = bare
	}
	if projectID == "" {
		projectID = generateProjectID()
	}

	contents := []any{}
	var sysParts []any

	for _, m := range req.Messages {
		switch m.Role {
		case model.RoleSystem:
			if t := partsText(m.Parts); t != "" {
				sysParts = append(sysParts, map[string]any{"text": t})
			}
		case model.RoleUser:
			contents = append(contents, map[string]any{"role": "user", "parts": userParts(m.Parts)})
		case model.RoleAssistant:
			parts := assistantParts(m.Parts)
			if len(parts) > 0 {
				contents = append(contents, map[string]any{"role": "model", "parts": parts})
			}
		case model.RoleTool:
			contents = append(contents, map[string]any{"role": "user", "parts": toolResultParts(m.Parts)})
		}
	}
	if len(contents) == 0 {
		contents = []any{map[string]any{"role": "user", "parts": []any{map[string]any{"text": ""}}}}
	}

	// Antigravity double-injects its default system prompt at the front.
	inject := []any{
		map[string]any{"text": antigravityDefaultSystem},
		map[string]any{"text": "Please ignore the following [ignore]" + antigravityDefaultSystem + "[/ignore]"},
	}
	systemInstruction := map[string]any{"role": "user", "parts": append(inject, sysParts...)}

	genCfg := map[string]any{"maxOutputTokens": maxTokens(req.MaxTokens)}
	if req.Temperature != nil {
		genCfg["temperature"] = *req.Temperature
	}
	genCfg["thinkingConfig"] = map[string]any{"thinkingBudget": 8192, "include_thoughts": true}

	request := map[string]any{
		"sessionId":         sessionID,
		"contents":          contents,
		"systemInstruction": systemInstruction,
		"generationConfig":  genCfg,
	}

	reverse := map[string]string{}
	if tools := buildTools(req.Tools, reverse); tools != nil {
		request["tools"] = tools
		request["toolConfig"] = map[string]any{"functionCallingConfig": map[string]any{"mode": "VALIDATED"}}
	}

	envelope := map[string]any{
		"project":     projectID,
		"model":       upstream,
		"userAgent":   "antigravity",
		"requestType": "agent",
		"requestId":   "agent-" + uuid(),
		"request":     request,
	}
	b, _ := json.Marshal(envelope)
	return b, reverse
}

func maxTokens(n int) int {
	if n <= 0 || n > maxOutputTokens {
		return maxOutputTokens
	}
	return n
}

// --- parts mapping ---

type partList = []model.Part

func partsText(parts partList) string {
	var b strings.Builder
	for _, p := range parts {
		if p.Type == "text" || p.Type == "" {
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

func userParts(parts partList) []any {
	out := []any{}
	for _, p := range parts {
		switch p.Type {
		case "image":
			if strings.HasPrefix(p.ImageURL, "data:") {
				mime, data := splitDataURL(p.ImageURL)
				out = append(out, map[string]any{"inlineData": map[string]any{"mime_type": mime, "data": data}})
			} else if p.ImageURL != "" {
				out = append(out, map[string]any{"fileData": map[string]any{"fileUri": p.ImageURL, "mimeType": "image/*"}})
			}
		default:
			out = append(out, map[string]any{"text": p.Text})
		}
	}
	if len(out) == 0 {
		out = []any{map[string]any{"text": ""}}
	}
	return out
}

func assistantParts(parts partList) []any {
	out := []any{}
	for _, p := range parts {
		switch p.Type {
		case "tool_use", "tool_call":
			id, name, args := toolCallFields(p)
			out = append(out, map[string]any{
				"thoughtSignature": defaultThinkingSignature,
				"functionCall":     map[string]any{"id": id, "name": name, "args": jsonObj(args)},
			})
		default:
			if p.Text != "" {
				out = append(out, map[string]any{"text": p.Text})
			}
		}
	}
	return out
}

func toolResultParts(parts partList) []any {
	out := []any{}
	for _, p := range parts {
		if p.ToolCallID == "" && p.Type != "tool_result" {
			continue
		}
		out = append(out, map[string]any{
			"functionResponse": map[string]any{
				"id":       p.ToolCallID,
				"name":     p.ToolName,
				"response": map[string]any{"result": p.Text},
			},
		})
	}
	if len(out) == 0 {
		out = []any{map[string]any{"text": ""}}
	}
	return out
}

func toolCallFields(p model.Part) (id, name, args string) {
	id, name = p.ToolCallID, p.ToolName
	if len(p.Raw) > 0 {
		var r struct {
			ID   string          `json:"id"`
			Name string          `json:"name"`
			Args json.RawMessage `json:"arguments"`
		}
		if json.Unmarshal(p.Raw, &r) == nil {
			if r.ID != "" {
				id = r.ID
			}
			if r.Name != "" {
				name = r.Name
			}
			if len(r.Args) > 0 {
				args = string(r.Args)
			}
		}
	}
	return id, name, args
}

// --- tools ---

var toolNameRe = regexp.MustCompile(`[^a-zA-Z0-9_.:\-]`)
var toolNameStartRe = regexp.MustCompile(`^[a-zA-Z_]`)

func buildTools(tools []model.Tool, reverse map[string]string) []any {
	if len(tools) == 0 {
		return nil
	}
	used := map[string]bool{}
	decls := []any{}
	for _, t := range tools {
		name := sanitizeToolName(t.Name, used)
		if name != t.Name {
			reverse[name] = t.Name
		}
		decls = append(decls, map[string]any{
			"name":        name,
			"description": t.Description,
			"parameters":  cleanSchema(t.Parameters),
		})
	}
	return []any{map[string]any{"functionDeclarations": decls}}
}

func sanitizeToolName(name string, used map[string]bool) string {
	s := toolNameRe.ReplaceAllString(name, "_")
	if s == "" || !toolNameStartRe.MatchString(s) {
		s = "_" + s
	}
	if len(s) > 64 {
		s = s[:64]
	}
	base := s
	for i := 2; used[s]; i++ {
		s = fmt.Sprintf("%s_%d", base, i)
	}
	used[s] = true
	return s
}

// unsupportedSchemaKeys are JSON-schema keywords CloudCode rejects.
var unsupportedSchemaKeys = map[string]bool{
	"minLength": true, "maxLength": true, "exclusiveMinimum": true, "exclusiveMaximum": true,
	"pattern": true, "minItems": true, "maxItems": true, "format": true, "default": true,
	"examples": true, "$schema": true, "$defs": true, "definitions": true, "const": true,
	"$ref": true, "$comment": true, "additionalProperties": true, "propertyNames": true,
	"patternProperties": true, "enumDescriptions": true, "anyOf": true, "oneOf": true,
	"allOf": true, "not": true, "dependencies": true, "dependentSchemas": true,
	"dependentRequired": true, "title": true, "if": true, "then": true, "else": true,
	"contentMediaType": true, "contentEncoding": true,
}

// cleanSchema strips keywords CloudCode can't parse and normalizes the shape.
func cleanSchema(raw json.RawMessage) map[string]any {
	empty := map[string]any{"type": "object", "properties": map[string]any{}}
	if len(raw) == 0 {
		return withReasonPlaceholder(empty)
	}
	var s map[string]any
	if json.Unmarshal(raw, &s) != nil {
		return withReasonPlaceholder(empty)
	}
	cleaned := cleanNode(s)
	m, ok := cleaned.(map[string]any)
	if !ok {
		return withReasonPlaceholder(empty)
	}
	// Infer object when properties present but no type.
	if _, hasType := m["type"]; !hasType {
		if _, hasProps := m["properties"]; hasProps {
			m["type"] = "object"
		}
	}
	// Empty object → reason placeholder.
	if props, ok := m["properties"].(map[string]any); ok && len(props) == 0 {
		return withReasonPlaceholder(m)
	}
	return m
}

func cleanNode(v any) any {
	switch n := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, val := range n {
			if unsupportedSchemaKeys[k] || strings.HasPrefix(k, "x-") {
				continue
			}
			if k == "const" {
				// const → single-value enum (string).
				out["enum"] = []any{fmt.Sprintf("%v", val)}
				out["type"] = "string"
				continue
			}
			out[k] = cleanNode(val)
		}
		// Prune required to existing props.
		if req, ok := out["required"].([]any); ok {
			if props, ok := out["properties"].(map[string]any); ok {
				kept := []any{}
				for _, r := range req {
					if name, ok := r.(string); ok {
						if _, exists := props[name]; exists {
							kept = append(kept, name)
						}
					}
				}
				out["required"] = kept
			}
		}
		return out
	case []any:
		out := make([]any, len(n))
		for i, e := range n {
			out[i] = cleanNode(e)
		}
		return out
	default:
		return v
	}
}

func withReasonPlaceholder(m map[string]any) map[string]any {
	m["type"] = "object"
	m["properties"] = map[string]any{"reason": map[string]any{"type": "string", "description": "Reason for the call"}}
	m["required"] = []any{"reason"}
	return m
}

// --- small helpers ---

func splitDataURL(u string) (mime, data string) {
	// data:image/png;base64,XXXX
	rest := strings.TrimPrefix(u, "data:")
	if i := strings.Index(rest, ","); i >= 0 {
		head := rest[:i]
		data = rest[i+1:]
		mime = strings.TrimSuffix(head, ";base64")
	}
	if mime == "" {
		mime = "image/png"
	}
	return mime, data
}

func jsonObj(s string) any {
	if strings.TrimSpace(s) == "" {
		return map[string]any{}
	}
	var v any
	if json.Unmarshal([]byte(s), &v) == nil {
		return v
	}
	return map[string]any{}
}

func uuid() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
