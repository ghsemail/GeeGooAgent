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
	if len(workflow.PostMarketPerStockSteps()) == 0 {
		t.Fatal("post_market steps empty")
	}
}
