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
	result, _ := memory.DecideIntraday(ws)
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

func TestIntradayPerStockStepsUsesWorkingFrequency(t *testing.T) {
	w := memory.NewPreMarketWorking("s1", "intraday_stock")
	in := workflow.IntradayInput{
		Code: "00700.HK", StockName: "腾讯控股",
		BotID: "b1", BotName: "bot", BotType: "DCA",
		Frequency: "15m", TradeType: "信号买入",
	}
	workflow.SeedIntradayWorking(w, in)
	steps := workflow.IntradayPerStockStepsForWorking(w)
	foundBundle := false
	for _, step := range steps {
		if step.Tool == "get_hourly_analysis_bundle" {
			foundBundle = true
		}
	}
	if !foundBundle {
		t.Fatal("expected hourly bundle step for 15m frequency from working input")
	}
}

func TestIntradayPerStockStepsSkipsHourlyForLowFrequency(t *testing.T) {
	w := memory.NewPreMarketWorking("s1", "intraday_stock")
	in := workflow.DefaultIntradayInput()
	in.Frequency = "3m"
	workflow.SeedIntradayWorking(w, in)
	steps := workflow.IntradayPerStockStepsForWorking(w)
	for _, step := range steps {
		if step.Tool == "get_mcp_analysis" || step.Tool == "get_hourly_analysis_bundle" {
			t.Fatalf("unexpected hourly step %s for 3m frequency", step.Tool)
		}
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
