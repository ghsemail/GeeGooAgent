package workflow_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func TestBuildPostMarketReportContentUsesChangePct(t *testing.T) {
	w := memory.NewPreMarketWorking("s1", "postmarket_stock")
	w.Stocks["601766.SH"] = memory.StockWorkspace{
		Code:       "601766.SH",
		StockName:  "中国中车",
		ChangePct:  2.35,
		PreMarketResult: "long",
		PreMarketReportID: "abc123",
		HourlyPriceAnalysis: "| foo | bar |\n| --- | --- |\n| 1 | 2 |",
	}
	out := workflow.BuildPostMarketReportContent(w, "601766.SH")
	if !strings.Contains(out, "2.35%") {
		t.Fatalf("expected change pct in report: %s", out)
	}
	if strings.Contains(out, "session_bias=") {
		t.Fatalf("expected localized text, got raw enums: %s", out)
	}
	if strings.Contains(out, "| foo |") {
		t.Fatalf("expected hourly table stripped: %s", out)
	}
}

func TestExperienceSummaryDefaultLocalized(t *testing.T) {
	ws := memory.StockWorkspace{SessionBias: "bullish", BotType: "GRID"}
	out := workflow.ExperienceSummaryDefault(ws, "aligned")
	if strings.Contains(out, "session_bias=") || strings.Contains(out, "aligned") {
		t.Fatalf("expected localized summary: %s", out)
	}
	if !strings.Contains(out, "偏多") || !strings.Contains(out, "一致") {
		t.Fatalf("expected Chinese labels: %s", out)
	}
}
