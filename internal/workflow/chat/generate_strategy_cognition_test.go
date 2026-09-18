package chat

import "testing"

func TestIsAgentCognitionContent(t *testing.T) {
	if !isAgentCognitionContent("doc_type: strategy_agent_cognition\n## 一句话定位") {
		t.Fatal("expected cognition marker")
	}
	if isAgentCognitionContent("4 Hour MACD Forex Strategy Welcome to the forex market") {
		t.Fatal("old web snippet should not pass")
	}
}

func TestFirstCognitionHitPreviewSkipsStaleHits(t *testing.T) {
	hits := []any{
		map[string]any{"content": "4 Hour MACD Forex Strategy old web text"},
		map[string]any{"content": "doc_type: strategy_agent_cognition\n## 一句话定位\nAgent 策略认知"},
	}
	got := firstCognitionHitPreview(hits)
	if got == "" || !isAgentCognitionContent(got) {
		t.Fatalf("snippet=%q", got)
	}
	if containsSubstr(got, "Forex") {
		t.Fatalf("should skip stale hit, got %q", got)
	}
}
