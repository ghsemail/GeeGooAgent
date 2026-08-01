package chatprompt_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
)

func TestAnalysisRoutingEmbedded(t *testing.T) {
	t.Parallel()
	got := chatprompt.AnalysisRouting()
	for _, want := range []string{
		"技术分析",
		"指标分析",
		"基本面分析",
		"type=tech",
		"type=index",
		"type=fundamental",
		"禁止默认 MACD",
		"get_mcp_analysis",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("AnalysisRouting missing %q", want)
		}
	}
}

func TestToolRoutingIncludesAnalysisRouting(t *testing.T) {
	t.Parallel()
	got := chatprompt.ToolRouting()
	if !strings.Contains(got, "MCP 分析路由") {
		t.Fatal("ToolRouting should embed analysis-routing.md")
	}
}
