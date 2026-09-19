package chat

import (
	"fmt"
	"testing"
)

func TestIsAgentCognitionContent(t *testing.T) {
	if !isAgentCognitionContent("doc_type: strategy_agent_cognition\n## 一句话定位") {
		t.Fatal("expected agent cognition marker")
	}
	if isAgentCognitionContent("4 Hour MACD Forex Strategy Welcome to the forex market") {
		t.Fatal("forex web snippet should not pass")
	}
}

func TestCognitionVerifySnippetHitLoop(t *testing.T) {
	hits := []any{
		map[string]any{"content": "forex"},
		map[string]any{"content": "doc_type: strategy_agent_cognition\n## 一句话定位\n\n正确摘要"},
	}
	for i, raw := range hits {
		row, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("%d not map", i)
		}
		content := fmt.Sprint(row["content"])
		t.Logf("%d agent=%v content=%q", i, isAgentCognitionContent(content), content)
	}
}

func TestCognitionVerifySnippetPrefersAgentDoc(t *testing.T) {
	flow := &Flow{
		KnowledgeTitle: "4小时MACD市场节奏 · 策略认知",
		CatalogLabel:   "4小时MACD市场节奏",
		KBDraft:        "# 4小时MACD市场节奏 · Agent 策略认知\n\n## 一句话定位\n\n测试摘要",
	}
	hits := []any{
		map[string]any{"content": "4 Hour MACD Forex Strategy Welcome to the forex market"},
		map[string]any{"content": "doc_type: strategy_agent_cognition\n## 一句话定位\n\n正确摘要"},
	}
	got := cognitionVerifySnippet(hits, flow)
	if got == "" || !isAgentCognitionContent(got) {
		t.Fatalf("snippet=%q", got)
	}
}

func TestCognitionVerifySnippetFallsBackToDraft(t *testing.T) {
	flow := &Flow{
		KBDraft: "doc_type: strategy_agent_cognition\n# Demo · Agent 策略认知\n\n## 一句话定位\n\n草稿摘要",
	}
	got := cognitionVerifySnippet([]any{
		map[string]any{"content": "4 Hour MACD Forex Strategy"},
	}, flow)
	if got == "" || !containsSubstr(got, "一句话定位") {
		t.Fatalf("snippet=%q", got)
	}
}
