package workflow_test

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/report"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

type countingStockPremarketSynth struct {
	calls atomic.Int32
}

func (c *countingStockPremarketSynth) Synthesize(ctx context.Context, ws memory.StockWorkspace, ev []memory.EvidenceRef, mc memory.MarketContext) (report.SynthesisResult, error) {
	return report.SynthesisResult{}, context.Canceled
}

func (c *countingStockPremarketSynth) SynthesizeMarket(ctx context.Context, market, draft string, mc memory.MarketContext, ev []memory.EvidenceRef, template string) (report.MarketSynthesisResult, error) {
	return report.MarketSynthesisResult{}, context.Canceled
}

func (c *countingStockPremarketSynth) SynthesizeStockPreMarket(ctx context.Context, ws memory.StockWorkspace, draft string, ev []memory.EvidenceRef, mc memory.MarketContext, marketReportSummary, template string) (report.StockPreMarketSynthesisResult, error) {
	c.calls.Add(1)
	return report.StockPreMarketSynthesisResult{
		Report:     "## 市场背景\n\nLLM 个股正文",
		Result:     "long",
		Confidence: "high",
		Reason:     strings.Repeat("LLM 综合理由引用证据 ", 12),
		Suggestion: "buy",
		Summary:    "LLM 摘要",
	}, nil
}

func TestStockPremarketSynthesisCachedAcrossSaveAndCreate(t *testing.T) {
	synth := &countingStockPremarketSynth{}
	ctx := workflow.ContextWithSynthesizer(context.Background(), synth)
	w := &memory.PreMarketWorking{
		BotCodes: []memory.BotStock{{Code: "00700.HK", StockName: "腾讯控股"}},
		Stocks: map[string]memory.StockWorkspace{
			"00700.HK": {Code: "00700.HK", StockName: "腾讯控股", Attitude: "bullish"},
		},
	}
	if _, err := workflow.BuildReportContent(ctx, w, "00700.HK"); err != nil {
		t.Fatal(err)
	}
	if _, err := workflow.BuildCreateReportArgsContext(ctx, w, "00700.HK"); err != nil {
		t.Fatal(err)
	}
	if got := synth.calls.Load(); got != 1 {
		t.Fatalf("expected one LLM synthesis call for save+create, got %d", got)
	}
	ws := w.Stocks["00700.HK"]
	if !ws.StockPremarketSynthesized || ws.StockPremarketReportBody == "" {
		t.Fatalf("expected synthesized bundle cached on working memory: %+v", ws)
	}
}
