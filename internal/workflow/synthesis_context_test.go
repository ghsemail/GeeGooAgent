package workflow_test

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/report"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

type contextCheckingSynthesizer struct {
	key  any
	want string
	got  context.Context
}

func (c *contextCheckingSynthesizer) Synthesize(ctx context.Context, ws memory.StockWorkspace, ev []memory.EvidenceRef, mc memory.MarketContext) (report.SynthesisResult, error) {
	c.got = ctx
	return report.SynthesisResult{
		Reason: "reason " + stringRepeat("x", 80), Suggestion: "hold", Summary: "ok",
	}, nil
}

func (c *contextCheckingSynthesizer) SynthesizeStockPreMarket(ctx context.Context, ws memory.StockWorkspace, draft string, ev []memory.EvidenceRef, mc memory.MarketContext, marketReportSummary, template string) (report.StockPreMarketSynthesisResult, error) {
	c.got = ctx
	return report.StockPreMarketSynthesisResult{
		Report: "## 市场背景\n\nctx ok", Result: "neutral", Confidence: "medium",
		Reason: "reason " + stringRepeat("x", 80), Suggestion: "hold", Summary: "ok",
	}, nil
}

func stringRepeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

func TestBuildCreateReportArgsPassesContextToSynthesis(t *testing.T) {
	key, want := "run-context", "run-context"
	ctx := context.WithValue(context.Background(), key, want)
	ctx = workflow.ContextWithSynthesizer(ctx, &contextCheckingSynthesizer{key: key, want: want})
	w := &memory.PreMarketWorking{
		BotCodes: []memory.BotStock{{Code: "00700.HK"}},
		Stocks:   map[string]memory.StockWorkspace{"00700.HK": {Code: "00700.HK", Attitude: "neutral"}},
	}
	args, err := workflow.BuildCreateReportArgsContext(ctx, w, "00700.HK")
	if err != nil {
		t.Fatal(err)
	}
	if args["suggestion"] != "hold" {
		t.Fatalf("args=%v", args)
	}
}
