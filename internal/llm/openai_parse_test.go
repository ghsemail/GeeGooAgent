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
