package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestThinkingOffMiniMaxM3SendsDisabledAndReasoningSplit(t *testing.T) {
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &captured)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(OpenAIOptions{
		Model: "MiniMax-M3", APIKey: "sk", BaseURL: srv.URL, ThinkingEnabled: true,
	})
	ctx := WithCallMeta(context.Background(), CallMeta{ThinkingOff: true})
	_, err := p.Chat(ctx, []Message{{Role: RoleUser, Content: "hi"}}, nil, 0, 64)
	if err != nil {
		t.Fatal(err)
	}
	th, _ := captured["thinking"].(map[string]any)
	if th["type"] != "disabled" {
		t.Fatalf("thinking=%v want disabled (omit leaves adaptive ON per MiniMax docs)", captured["thinking"])
	}
	if captured["reasoning_split"] != true {
		t.Fatalf("reasoning_split=%v", captured["reasoning_split"])
	}
	if _, ok := captured["reasoning_effort"]; ok {
		t.Fatalf("should not send reasoning_effort when disabled: %v", captured)
	}
}

func TestThinkingOffDeepSeekFlashSendsDisabled(t *testing.T) {
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &captured)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(OpenAIOptions{
		Model: "deepseek-v4-flash", APIKey: "sk", BaseURL: srv.URL, ThinkingEnabled: true,
	})
	ctx := WithCallMeta(context.Background(), CallMeta{ThinkingOff: true})
	if _, err := p.Chat(ctx, []Message{{Role: RoleUser, Content: "hi"}}, nil, 0, 64); err != nil {
		t.Fatal(err)
	}
	th, _ := captured["thinking"].(map[string]any)
	if th["type"] != "disabled" {
		t.Fatalf("thinking=%v want disabled for compose flash", captured["thinking"])
	}
}

func TestThinkingOffOmitsEnabledBlockForNonMiniMax(t *testing.T) {
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &captured)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(OpenAIOptions{
		Model: "gpt-4o-mini", APIKey: "sk", BaseURL: srv.URL, ThinkingEnabled: true,
	})
	ctx := WithCallMeta(context.Background(), CallMeta{ThinkingOff: true})
	if _, err := p.Chat(ctx, []Message{{Role: RoleUser, Content: "hi"}}, nil, 0, 64); err != nil {
		t.Fatal(err)
	}
	if _, ok := captured["thinking"]; ok {
		t.Fatalf("thinking should be omitted: %v", captured)
	}
}

func TestThinkingOnMiniMaxWithoutThinkingOffOmitsExplicitDisabled(t *testing.T) {
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &captured)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(OpenAIOptions{
		Model: "MiniMax-M3", APIKey: "sk", BaseURL: srv.URL, ThinkingEnabled: false,
	})
	if _, err := p.Chat(context.Background(), []Message{{Role: RoleUser, Content: "hi"}}, nil, 0, 64); err != nil {
		t.Fatal(err)
	}
	if _, ok := captured["thinking"]; ok {
		t.Fatalf("provider thinking off: body should omit thinking (MiniMax default adaptive): %v", captured)
	}
}
