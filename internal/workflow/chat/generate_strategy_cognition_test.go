package chat

import (
	"fmt"
	"testing"
)

func TestIsStrategyArchiveContent(t *testing.T) {
	if !isStrategyArchiveContent("doc_type: strategy_agent_cognition\n## 一句话定位") {
		t.Fatal("expected strategy archive marker")
	}
	if !isStrategyArchiveContent("# Macd4H · 策略档案\n\n## 一句话定位\n\n摘要") {
		t.Fatal("expected 策略档案 heading")
	}
	if !isStrategyArchiveContent("# Macd4H · Agent 策略认知\n\n## 一句话定位\n\n摘要") {
		t.Fatal("expected legacy agent cognition heading")
	}
	if isStrategyArchiveContent("4 Hour MACD Forex Strategy Welcome to the forex market") {
		t.Fatal("forex web snippet should not pass")
	}
}

func TestFormatGenerateStrategyArchiveMessage(t *testing.T) {
	if got := FormatGenerateStrategyArchiveMessage("Macd4H"); got != "帮我生成 Macd4H 的策略档案" {
		t.Fatalf("got %q", got)
	}
	if got := FormatGenerateStrategyArchiveMessage(""); got != "帮我生成策略档案" {
		t.Fatalf("got %q", got)
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
		t.Logf("%d archive=%v content=%q", i, isStrategyArchiveContent(content), content)
	}
}

func TestCognitionVerifySnippetPrefersAgentDoc(t *testing.T) {
	flow := &Flow{
		KnowledgeTitle: "4小时MACD市场节奏 · 策略档案",
		CatalogLabel:   "4小时MACD市场节奏",
		KBDraft:        "# 4小时MACD市场节奏 · 策略档案\n\n## 一句话定位\n\n测试摘要",
	}
	hits := []any{
		map[string]any{"content": "4 Hour MACD Forex Strategy Welcome to the forex market"},
		map[string]any{"content": "doc_type: strategy_agent_cognition\n## 一句话定位\n\n正确摘要"},
	}
	got := cognitionVerifySnippet(hits, flow)
	if got == "" || !isStrategyArchiveContent(got) {
		t.Fatalf("snippet=%q", got)
	}
}

func TestCognitionVerifySnippetFallsBackToDraft(t *testing.T) {
	flow := &Flow{
		KBDraft: "doc_type: strategy_agent_cognition\n# Demo · 策略档案\n\n## 一句话定位\n\n草稿摘要",
	}
	got := cognitionVerifySnippet([]any{
		map[string]any{"content": "4 Hour MACD Forex Strategy"},
	}, flow)
	if got == "" || !containsSubstr(got, "一句话定位") {
		t.Fatalf("snippet=%q", got)
	}
}
