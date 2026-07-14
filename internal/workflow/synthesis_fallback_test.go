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
	return "", "", "", context.DeadlineExceeded
}

func TestBuildCreateReportArgsFallsBackOnSynthesisError(t *testing.T) {
	// NOT parallel: mutates package-level defaultSynthesizer.
	workflow.SetDefaultSynthesizer(&failingSynthesizer{})
	defer workflow.SetDefaultSynthesizer(nil)

	w := memory.NewPreMarketWorking("s1", "pre_market")
	trading := true
	w.IsTradingDay = &trading
	w.BotCodes = []memory.BotStock{{Code: "00700.HK", StockName: "腾讯控股", BotID: "b1", BotName: "bot", BotType: "DCA"}}
	w.Stocks["00700.HK"] = memory.StockWorkspace{
		Code: "00700.HK", StockName: "腾讯控股", BotID: "b1", BotName: "bot", BotType: "DCA",
		Status: "collecting", Attitude: "bullish",
	}
	args := workflow.BuildCreateReportArgs(w, "00700.HK")
	if args["result"] != "long" {
		t.Fatalf("result should be long from attitude, got %v", args["result"])
	}
	reason, _ := args["reason"].(string)
	if !strings.Contains(reason, "attitude is bullish") {
		t.Fatalf("reason should fall back to rule-based, got %q", reason)
	}
}

type happySynthesizer struct{}

func (h *happySynthesizer) Synthesize(ctx context.Context, ws memory.StockWorkspace, ev []memory.EvidenceRef, mc memory.MarketContext) (string, string, string, error) {
	return strings.Repeat("LLM 综合理由引用证据 ", 12), "buy", "LLM 摘要", nil
}

func TestBuildCreateReportArgsUsesSynthesisWhenSuccessful(t *testing.T) {
	// NOT parallel: mutates package-level defaultSynthesizer.
	workflow.SetDefaultSynthesizer(&happySynthesizer{})
	defer workflow.SetDefaultSynthesizer(nil)

	w := memory.NewPreMarketWorking("s2", "pre_market")
	trading := true
	w.IsTradingDay = &trading
	w.BotCodes = []memory.BotStock{{Code: "00700.HK", StockName: "腾讯", BotID: "b1", BotName: "bot", BotType: "DCA"}}
	w.Stocks["00700.HK"] = memory.StockWorkspace{Code: "00700.HK", StockName: "腾讯", BotID: "b1", BotName: "bot", BotType: "DCA", Attitude: "neutral"}
	args := workflow.BuildCreateReportArgs(w, "00700.HK")
	if args["result"] != "neutral" {
		t.Fatalf("result must stay rule-based, got %v", args["result"])
	}
	if args["suggestion"] != "buy" {
		t.Fatalf("suggestion should come from synthesis, got %v", args["suggestion"])
	}
	if args["summary"] != "LLM 摘要" {
		t.Fatalf("summary should come from synthesis, got %v", args["summary"])
	}
}
