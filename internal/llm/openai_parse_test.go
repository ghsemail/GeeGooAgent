package llm

import (
	"strings"
	"testing"
)

func TestParseOpenAIResponseRejectsInvalidToolArguments(t *testing.T) {
	_, err := parseOpenAIResponse([]byte(`{"choices":[{"message":{"tool_calls":[{"id":"call-1","function":{"name":"create_pre_market_report","arguments":"{not-json"}}]}}]}`), "mock")
	if err == nil {
		t.Fatal("expected invalid tool arguments to be rejected")
	}
	if !strings.Contains(err.Error(), "create_pre_market_report") {
		t.Fatalf("error should identify the unsafe tool call: %v", err)
	}
}

func TestParseToolArgumentsRejectsNonObject(t *testing.T) {
	_, err := ParseToolArguments([]any{"unexpected"})
	if err == nil {
		t.Fatal("expected non-object tool arguments to be rejected")
	}
}

func TestParseOpenAIResponseCoercesNullContentAndFinishReason(t *testing.T) {
	resp, err := parseOpenAIResponse([]byte(`{
		"choices":[{"finish_reason":"length","message":{"content":null,"reasoning_content":"think"}}],
		"usage":{"prompt_tokens":10,"completion_tokens":20}
	}`), "deepseek-v4-flash")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "" {
		t.Fatalf("content=%q", resp.Content)
	}
	if resp.ReasoningContent != "think" {
		t.Fatalf("reasoning=%q", resp.ReasoningContent)
	}
	if resp.FinishReason != "length" {
		t.Fatalf("finish=%q", resp.FinishReason)
	}
}

func TestParseOpenAIResponseMultipartContent(t *testing.T) {
	resp, err := parseOpenAIResponse([]byte(`{
		"choices":[{"finish_reason":"stop","message":{"content":[{"type":"text","text":"hello"}]}}]
	}`), "m")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "hello" {
		t.Fatalf("content=%q", resp.Content)
	}
}
