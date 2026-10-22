package llm

import (
	"context"
	"encoding/json"
)

type composeTraceKey struct{}

// ComposeRequestTrace captures the outbound chat/completions body shape for diag_compose diagnostics.
type ComposeRequestTrace struct {
	BaseURL       string  `json:"base_url,omitempty"`
	Model         string  `json:"model,omitempty"`
	Stream        bool    `json:"stream"`
	MaxTokens     int     `json:"max_tokens,omitempty"`
	Temperature   float64 `json:"temperature,omitempty"`
	ThinkingJSON  string  `json:"thinking_json,omitempty"`
	SystemChars   int     `json:"system_chars,omitempty"`
	UserChars     int     `json:"user_chars,omitempty"`
	ThinkingOff   bool    `json:"thinking_off,omitempty"`
}

// WithComposeRequestTrace attaches a trace record to fill during the next provider HTTP call.
func WithComposeRequestTrace(ctx context.Context, trace *ComposeRequestTrace) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if trace == nil {
		return ctx
	}
	return context.WithValue(ctx, composeTraceKey{}, trace)
}

func composeRequestTraceFrom(ctx context.Context) *ComposeRequestTrace {
	if ctx == nil {
		return nil
	}
	if v, ok := ctx.Value(composeTraceKey{}).(*ComposeRequestTrace); ok {
		return v
	}
	return nil
}

func MessageCharCounts(messages []Message) (systemChars, userChars int) {
	for _, m := range messages {
		n := len(m.Content)
		switch m.Role {
		case RoleSystem:
			systemChars += n
		case RoleUser:
			userChars += n
		}
	}
	return systemChars, userChars
}

func fillComposeTraceFromBody(ctx context.Context, baseURL, model string, body map[string]any, systemChars, userChars int) {
	tr := composeRequestTraceFrom(ctx)
	if tr == nil {
		return
	}
	tr.BaseURL = baseURL
	tr.Model = model
	if v, ok := body["stream"].(bool); ok {
		tr.Stream = v
	}
	if v, ok := body["max_tokens"].(int); ok {
		tr.MaxTokens = v
	} else if v, ok := body["max_tokens"].(float64); ok {
		tr.MaxTokens = int(v)
	}
	if v, ok := body["temperature"].(float64); ok {
		tr.Temperature = v
	}
	if th, ok := body["thinking"]; ok {
		b, _ := json.Marshal(th)
		tr.ThinkingJSON = string(b)
	}
	meta := CallMetaFrom(ctx)
	tr.ThinkingOff = meta.ThinkingOff
	tr.SystemChars = systemChars
	tr.UserChars = userChars
}
