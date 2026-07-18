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

func TestParseToolArgumentsConcatenatedObjects(t *testing.T) {
	got, err := ParseToolArguments(`{"code":"TSLA","period":"daily"}{"code":"TSLA","period":"daily"}`)
	if err != nil {
		t.Fatal(err)
	}
	if got["code"] != "TSLA" || got["period"] != "daily" {
		t.Fatalf("got %#v", got)
	}
}

func TestStreamToolAccResetsOnResentObject(t *testing.T) {
	acc := &streamToolAcc{}
	acc.appendArgs(`{"code":"TSLA"}`)
	acc.appendArgs(`{"code":"TSLA","period":"daily"}`)
	got, err := ParseToolArguments(acc.Arguments.String())
	if err != nil {
		t.Fatal(err)
	}
	if got["period"] != "daily" {
		t.Fatalf("got %#v from %q", got, acc.Arguments.String())
	}
}

func TestParseOpenAIStreamResentToolArguments(t *testing.T) {
	sse := strings.Join([]string{
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","function":{"name":"get_mcp_analysis","arguments":"{\"code\":\"TSLA\"}"}}]}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"code\":\"TSLA\",\"period\":\"daily\"}"}}]},"finish_reason":"tool_calls"}]}`,
		`data: [DONE]`,
		"",
	}, "\n")
	resp, err := parseOpenAIStream(strings.NewReader(sse), "m", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "get_mcp_analysis" {
		t.Fatalf("tools=%+v", resp.ToolCalls)
	}
	if resp.ToolCalls[0].Arguments["period"] != "daily" {
		t.Fatalf("args=%v", resp.ToolCalls[0].Arguments)
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

func TestParseOpenAIResponseCacheUsage(t *testing.T) {
	resp, err := parseOpenAIResponse([]byte(`{
		"choices":[{"finish_reason":"stop","message":{"content":"ok"}}],
		"usage":{"prompt_tokens":100,"completion_tokens":10,"prompt_cache_hit_tokens":80,"prompt_cache_miss_tokens":20}
	}`), "deepseek-v4-flash")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Usage.PromptCacheHitTokens != 80 || resp.Usage.PromptCacheMissTokens != 20 {
		t.Fatalf("usage=%+v", resp.Usage)
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
