package workflow_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

// failingSynthesizer always errors, forcing the rule-based fallback.
type failingSynthesizer struct{}

func (f *failingSynthesizer) Synthesize(ctx context.Context, ws memory.StockWorkspace, ev []memory.EvidenceRef, mc memory.MarketContext) (string, string, string, error) {
	return "", "", "", context.Canceled
}

func TestBuildCreateReportArgsFallsBackOnSynthesisError(t *testing.T) {
	ctx := workflow.ContextWithSynthesizer(context.Background(), &failingSynthesizer{})
	w := &memory.PreMarketWorking{
		BotCodes: []memory.BotStock{{Code: "00700.HK", StockName: "腾讯控股"}},
		Stocks: map[string]memory.StockWorkspace{
			"00700.HK": {Code: "00700.HK", StockName: "腾讯控股", Attitude: "bullish"},
		},
	}
	args := workflow.BuildCreateReportArgsContext(ctx, w, "00700.HK")
	if args["result"] != "long" {
		t.Fatalf("result=%v want long (rule-based)", args["result"])
	}
}

type happySynthesizer struct{}

func (h *happySynthesizer) Synthesize(ctx context.Context, ws memory.StockWorkspace, ev []memory.EvidenceRef, mc memory.MarketContext) (string, string, string, error) {
	return strings.Repeat("LLM 综合理由引用证据 ", 12), "buy", "LLM 摘要", nil
}

func TestBuildCreateReportArgsUsesSynthesisWhenSuccessful(t *testing.T) {
	ctx := workflow.ContextWithSynthesizer(context.Background(), &happySynthesizer{})
	w := &memory.PreMarketWorking{
		BotCodes: []memory.BotStock{{Code: "00700.HK"}},
		Stocks: map[string]memory.StockWorkspace{
			"00700.HK": {Code: "00700.HK", Attitude: "bullish"},
		},
	}
	args := workflow.BuildCreateReportArgsContext(ctx, w, "00700.HK")
	if args["summary"] != "LLM 摘要" {
		t.Fatalf("summary=%v", args["summary"])
	}
}
