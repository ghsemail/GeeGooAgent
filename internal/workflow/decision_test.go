package workflow_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func TestDecideIntradaySellWithoutPosition(t *testing.T) {
	ws := memory.StockWorkspace{
		TradeType: "信号卖出", BotType: "DCA", HasPosition: false,
	}
	result, _ := workflow.DecideIntraday(ws)
	if result != "hold" {
		t.Fatalf("expected hold, got %s", result)
	}
}

func TestVsPreMarketAligned(t *testing.T) {
	if workflow.VsPreMarket("long", "bullish") != "aligned" {
		t.Fatal("expected aligned")
	}
	if workflow.VsPreMarket("long", "bearish") != "contradicted" {
		t.Fatal("expected contradicted")
	}
}

func TestSessionBiasFromChangePct(t *testing.T) {
	if workflow.SessionBiasFromChangePct(2) != "bullish" {
		t.Fatal("expected bullish")
	}
	if workflow.SessionBiasFromChangePct(-2) != "bearish" {
		t.Fatal("expected bearish")
	}
}

func TestSeedIntradayWorking(t *testing.T) {
	w := memory.NewPreMarketWorking("s1", "intraday")
	in := workflow.DefaultIntradayInput()
	workflow.SeedIntradayWorking(w, in)
	if len(w.BotCodes) != 1 || w.Stocks[in.Code].Status != "collecting" {
		t.Fatalf("unexpected seed state: %+v", w)
	}
}

func TestIntradayPerStockStepsNonEmpty(t *testing.T) {
	if len(workflow.IntradayPerStockSteps()) == 0 {
		t.Fatal("intraday steps empty")
	}
}

func TestPostMarketPerStockStepsNonEmpty(t *testing.T) {
	steps := workflow.PostMarketPerStockSteps()
	if len(steps) == 0 {
		t.Fatal("postmarket_stock steps empty")
	}
	if steps[1].Tool != "get_hourly_analysis_bundle" {
		t.Fatalf("expected hourly bundle as step 2, got %s", steps[1].Tool)
	}
}

func TestLegacyHourlyStepsSatisfyBundleResume(t *testing.T) {
	w := memory.NewPreMarketWorking("s1", "postmarket_stock")
	for _, key := range []string{"hourly_price_analysis", "hourly_signal_analysis", "hourly_kline_analysis"} {
		workflow.MarkStepCompleteForTest(w, key)
	}
	if !workflow.IsStepCompleteForTest(w, "hourly_analysis_bundle") {
		t.Fatal("legacy hourly steps should satisfy bundle resume")
	}
}
