package workflow_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/report"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func TestBuildPostMarketReportContentUsesChangePct(t *testing.T) {
	w := memory.NewPreMarketWorking("s1", "postmarket_stock")
	w.Stocks["601766.SH"] = memory.StockWorkspace{
		Code:              "601766.SH",
		StockName:         "中国中车",
		ChangePct:         2.35,
		PreMarketResult:   "long",
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
	if strings.Contains(out, "关联盘前报告") || strings.Contains(out, "abc123") {
		t.Fatalf("premarket report id must not appear in对照 narrative: %s", out)
	}
	if !strings.Contains(out, "盘前判断为") || !strings.Contains(out, "对照结论") {
		t.Fatalf("expected对照 narrative without id: %s", out)
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

type postMarketSummaryMock struct{}

func (p *postMarketSummaryMock) Synthesize(
	ctx context.Context, ws memory.StockWorkspace, ev []memory.EvidenceRef, mc memory.MarketContext,
) (report.SynthesisResult, error) {
	return report.SynthesisResult{}, nil
}

func (p *postMarketSummaryMock) SynthesizePostMarketSummaries(
	ctx context.Context,
	ws memory.StockWorkspace,
	draft string,
	sessionBias, vsPreMarket string,
	ruleMarket, ruleTrade, ruleExperience string,
) (report.PostMarketSynthesisResult, error) {
	return report.PostMarketSynthesisResult{
		MarketSummary:     "LLM行情摘要：腾讯收涨震荡，小时级空方占优但跌幅有限，盘面中性。",
		TradeSummary:      "LLM交易摘要：网格当日无成交，仓位与策略状态维持。",
		ExperienceSummary: "LLM经验摘要：盘前中性与收盘一致，次日应关注网格挂单与波动匹配度。",
		Summary:           "LLM一句话：腾讯中性收涨，网格无成交。",
	}, nil
}

func TestBuildCreateStockPostmarketReportArgsUsesLLMSummaries(t *testing.T) {
	w := memory.NewPreMarketWorking("s1", "postmarket_stock")
	w.Stocks["00700.HK"] = memory.StockWorkspace{
		Code: "00700.HK", StockName: "腾讯控股", BotType: "GRID",
		ChangePct: 0.54, SessionBias: "neutral", PreMarketResult: "neutral",
	}
	ctx := workflow.ContextWithSynthesizer(context.Background(), &postMarketSummaryMock{})
	args := workflow.BuildCreateStockPostmarketReportArgs(ctx, w, "00700.HK")
	if args["market_summary"] != "LLM行情摘要：腾讯收涨震荡，小时级空方占优但跌幅有限，盘面中性。" {
		t.Fatalf("market_summary=%v", args["market_summary"])
	}
	if args["summary"] != "LLM一句话：腾讯中性收涨，网格无成交。" {
		t.Fatalf("summary=%v", args["summary"])
	}
	if !strings.Contains(fmt.Sprint(args["experience_summary"]), "LLM经验摘要") {
		t.Fatalf("experience_summary=%v", args["experience_summary"])
	}
}
