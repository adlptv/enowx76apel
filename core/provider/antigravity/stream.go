package antigravity

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/enowdev/enowx/core/model"
)

// antigravityStream parses the CloudCode SSE stream (a Gemini
// GenerateContentResponse nested under `response`) into normalized events.
type antigravityStream struct {
	resp    *http.Response
	sc      *bufio.Scanner
	reverse map[string]string

	done     bool
	toolIdx  int
	sawTool  bool
	pending  []model.Event
}

func newAntigravityStream(resp *http.Response, reverse map[string]string) *antigravityStream {
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	sc.Split(splitSSE)
	return &antigravityStream{resp: resp, sc: sc, reverse: reverse}
}

func splitSSE(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
		return i + 2, data[:i], nil
	}
	if i := bytes.Index(data, []byte("\r\n\r\n")); i >= 0 {
		return i + 4, data[:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// geminiResponse is the payload (already unwrapped from `response`).
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int64 `json:"promptTokenCount"`
		CandidatesTokenCount int64 `json:"candidatesTokenCount"`
		ThoughtsTokenCount   int64 `json:"thoughtsTokenCount"`
		TotalTokenCount      int64 `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

type geminiPart struct {
	Text             string          `json:"text"`
	Thought          bool            `json:"thought"`
	ThoughtSignature string          `json:"thoughtSignature"`
	FunctionCall     *struct {
		Name string          `json:"name"`
		Args json.RawMessage `json:"args"`
	} `json:"functionCall"`
	InlineData *struct {
		MimeType string `json:"mimeType"`
		Data     string `json:"data"`
	} `json:"inlineData"`
}

func (s *antigravityStream) Recv() (model.Event, error) {
	if len(s.pending) > 0 {
		ev := s.pending[0]
		s.pending = s.pending[1:]
		return ev, nil
	}
	if s.done {
		return model.Event{}, io.EOF
	}
	for s.sc.Scan() {
		data := sseData(s.sc.Bytes())
		if len(data) == 0 || bytes.Equal(bytes.TrimSpace(data), []byte("[DONE]")) {
			continue
		}
		// Unwrap {response: {...}} (CloudCode nests everything under response).
		var wrap struct {
			Response json.RawMessage `json:"response"`
		}
		_ = json.Unmarshal(data, &wrap)
		payload := data
		if len(wrap.Response) > 0 {
			payload = wrap.Response
		}
		var gr geminiResponse
		if json.Unmarshal(payload, &gr) != nil || len(gr.Candidates) == 0 {
			continue
		}
		if ev, ok := s.handle(&gr); ok {
			return ev, nil
		}
	}
	s.done = true
	return model.Event{Type: model.EventDone}, nil
}

func (s *antigravityStream) handle(gr *geminiResponse) (model.Event, bool) {
	cand := gr.Candidates[0]
	var text, reasoning string
	var toolCalls []model.ToolCallDelta

	for _, p := range cand.Content.Parts {
		switch {
		case p.FunctionCall != nil:
			name := p.FunctionCall.Name
			if orig, ok := s.reverse[name]; ok {
				name = orig
			}
			idx := s.toolIdx
			s.toolIdx++
			s.sawTool = true
			args := "{}"
			if len(p.FunctionCall.Args) > 0 {
				args = string(p.FunctionCall.Args)
			}
			toolCalls = append(toolCalls, model.ToolCallDelta{Index: idx, ID: name + "-" + itoa(idx), Name: name, ArgsDelta: args})
		case p.InlineData != nil:
			// enowx has no image-delta event; surface as a data-URL in the text.
			text += "\n![image](data:" + p.InlineData.MimeType + ";base64," + p.InlineData.Data + ")\n"
		case p.Thought || p.ThoughtSignature != "":
			reasoning += p.Text
		default:
			text += p.Text
		}
	}

	// Final chunk carries usage + finishReason.
	if cand.FinishReason != "" {
		finish := "stop"
		if s.sawTool {
			finish = "tool_calls"
		}
		var usage *model.Usage
		if gr.UsageMetadata.TotalTokenCount > 0 {
			comp := gr.UsageMetadata.CandidatesTokenCount + gr.UsageMetadata.ThoughtsTokenCount
			usage = &model.Usage{PromptTokens: gr.UsageMetadata.PromptTokenCount, CompletionTokens: comp}
		}
		// Emit any content first, then a finish event, then done.
		s.pending = append(s.pending, model.Event{Type: model.EventDelta, FinishReason: finish, Usage: usage})
		s.pending = append(s.pending, model.Event{Type: model.EventDone})
		if text != "" || reasoning != "" || len(toolCalls) > 0 {
			return model.Event{Type: model.EventDelta, Text: text, Reasoning: reasoning, ToolCalls: toolCalls}, true
		}
		ev := s.pending[0]
		s.pending = s.pending[1:]
		return ev, true
	}

	if text != "" || reasoning != "" || len(toolCalls) > 0 {
		return model.Event{Type: model.EventDelta, Text: text, Reasoning: reasoning, ToolCalls: toolCalls}, true
	}
	return model.Event{}, false
}

func (s *antigravityStream) Close() error { return s.resp.Body.Close() }

func sseData(block []byte) []byte {
	var buf bytes.Buffer
	for _, line := range bytes.Split(block, []byte("\n")) {
		line = bytes.TrimRight(line, "\r")
		if bytes.HasPrefix(line, []byte("data:")) {
			if buf.Len() > 0 {
				buf.WriteByte('\n')
			}
			buf.Write(bytes.TrimSpace(line[5:]))
		}
	}
	return buf.Bytes()
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}
