package workflow_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/report"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

// failingSynthesizer always errors; premarket report build must fail (no rule-based fallback).
type failingSynthesizer struct{}

func (f *failingSynthesizer) Synthesize(ctx context.Context, ws memory.StockWorkspace, ev []memory.EvidenceRef, mc memory.MarketContext) (report.SynthesisResult, error) {
	return report.SynthesisResult{}, context.Canceled
}

func (f *failingSynthesizer) SynthesizeMarket(ctx context.Context, market, draft string, mc memory.MarketContext, ev []memory.EvidenceRef, template string) (report.MarketSynthesisResult, error) {
	return report.MarketSynthesisResult{}, context.Canceled
}

func (f *failingSynthesizer) SynthesizeStockPreMarket(ctx context.Context, ws memory.StockWorkspace, draft string, ev []memory.EvidenceRef, mc memory.MarketContext, marketReportSummary, template string) (report.StockPreMarketSynthesisResult, error) {
	return report.StockPreMarketSynthesisResult{}, context.Canceled
}

func TestBuildCreateReportArgsFailsOnSynthesisError(t *testing.T) {
	ctx := workflow.ContextWithSynthesizer(context.Background(), &failingSynthesizer{})
	w := &memory.PreMarketWorking{
		BotCodes: []memory.BotStock{{Code: "00700.HK", StockName: "腾讯控股"}},
		Stocks: map[string]memory.StockWorkspace{
			"00700.HK": {Code: "00700.HK", StockName: "腾讯控股", Attitude: "bullish"},
		},
	}
	_, err := workflow.BuildCreateReportArgsContext(ctx, w, "00700.HK")
	if err == nil {
		t.Fatal("expected error when stock premarket synthesis fails")
	}
}

type happySynthesizer struct{}

func (h *happySynthesizer) Synthesize(ctx context.Context, ws memory.StockWorkspace, ev []memory.EvidenceRef, mc memory.MarketContext) (report.SynthesisResult, error) {
	return report.SynthesisResult{
		SuggestedResult: "long", SuggestedConfidence: "high",
		Reason: strings.Repeat("LLM 综合理由引用证据 ", 12), Suggestion: "buy", Summary: "LLM 摘要",
	}, nil
}

func TestBuildCreateReportArgsUsesSynthesisWhenSuccessful(t *testing.T) {
	ctx := workflow.ContextWithSynthesizer(context.Background(), &happySynthesizer{})
	w := &memory.PreMarketWorking{
		BotCodes: []memory.BotStock{{Code: "00700.HK"}},
		Stocks: map[string]memory.StockWorkspace{
			"00700.HK": {Code: "00700.HK", Attitude: "bullish"},
		},
	}
	args, err := workflow.BuildCreateReportArgsContext(ctx, w, "00700.HK")
	if err != nil {
		t.Fatal(err)
	}
	if args["summary"] != "LLM 摘要" {
		t.Fatalf("summary=%v", args["summary"])
	}
	if !strings.Contains(args["report"].(string), "LLM 个股正文") {
		t.Fatalf("report=%v", args["report"])
	}
}

func TestBuildCreateMarketReportArgsFailsOnSynthesisError(t *testing.T) {
	ctx := workflow.ContextWithSynthesizer(context.Background(), &failingSynthesizer{})
	w := memory.NewPreMarketWorking("s1", "premarket_market")
	w.Market = "CN"
	w.MarketContext.IndicesDone = true
	w.MarketContext.MarketNewsDone = true
	w.MarketContext.IndexAnalysisRefs = map[string]string{"000001.SH": "偏强"}
	w.MarketContext.MarketNews = map[string]string{"CN": "政策偏暖"}
	_, err := workflow.BuildCreateMarketReportArgsContext(ctx, w, "CN")
	if err == nil {
		t.Fatal("expected error when market premarket synthesis fails")
	}
}

func (h *happySynthesizer) SynthesizeMarket(ctx context.Context, market, draft string, mc memory.MarketContext, ev []memory.EvidenceRef, template string) (report.MarketSynthesisResult, error) {
	return report.MarketSynthesisResult{
		Report:     "# A股 市场盘前报告\n\nLLM 正文",
		Result:     "long",
		Confidence: "high",
		Summary:    "LLM 市场摘要",
	}, nil
}

func (h *happySynthesizer) SynthesizeStockPreMarket(ctx context.Context, ws memory.StockWorkspace, draft string, ev []memory.EvidenceRef, mc memory.MarketContext, marketReportSummary, template string) (report.StockPreMarketSynthesisResult, error) {
	return report.StockPreMarketSynthesisResult{
		Report:     "## 市场背景\n\nLLM 个股正文",
		Result:     "long",
		Confidence: "high",
		Reason:     strings.Repeat("LLM 综合理由引用证据 ", 12),
		Suggestion: "buy",
		Summary:    "LLM 摘要",
	}, nil
}

func TestBuildCreateMarketReportArgsUsesMarketSynthesisWhenSuccessful(t *testing.T) {
	ctx := workflow.ContextWithSynthesizer(context.Background(), &happySynthesizer{})
	w := memory.NewPreMarketWorking("s2", "premarket_market")
	w.Market = "CN"
	w.MarketContext.IndicesDone = true
	w.MarketContext.MarketNewsDone = true
	w.MarketContext.IndexAnalysisRefs = map[string]string{"000001.SH": "偏强"}
	w.MarketContext.MarketNews = map[string]string{"CN": "政策偏暖"}
	args, err := workflow.BuildCreateMarketReportArgsContext(ctx, w, "CN")
	if err != nil {
		t.Fatal(err)
	}
	if args["summary"] != "LLM 市场摘要" {
		t.Fatalf("summary=%v", args["summary"])
	}
	if args["result"] != "long" || args["confidence"] != "high" {
		t.Fatalf("result=%v confidence=%v", args["result"], args["confidence"])
	}
	if !strings.Contains(args["report"].(string), "LLM 正文") {
		t.Fatalf("report=%v", args["report"])
	}
}
