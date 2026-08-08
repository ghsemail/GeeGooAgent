package workflow_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func TestOptionalStepIndexPrefix(t *testing.T) {
	if !workflow.OptionalStepForTest(workflow.Step{Name: "index_^DJI.US", Tool: "get_mcp_analysis"}) {
		t.Fatal("expected index step to be optional")
	}
	if !workflow.OptionalStepForTest(workflow.Step{Name: "market_news_cn", Tool: "fetch_market_news"}) {
		t.Fatal("expected market news step to be optional")
	}
	if workflow.OptionalStepForTest(workflow.Step{Name: "weekly_analysis", Tool: "get_mcp_analysis"}) {
		t.Fatal("stock analysis must not be optional")
	}
}

func TestFinalizePhaseAZeroBots(t *testing.T) {
	trading := true
	w := memory.NewPreMarketWorking("s1", "postmarket_stock")
	w.IsTradingDay = &trading
	w.Phase = "phase_a"
	workflow.FinalizePhaseAForTest(w)
	if w.Phase != "done" {
		t.Fatalf("phase=%s want done", w.Phase)
	}
}

func TestRecordIndexSkipCompletesIndices(t *testing.T) {
	w := memory.NewPreMarketWorking("s1", "premarket_market")
	for _, code := range []string{"^DJI.US", "^IXIC.US", "000001.SH", "399001.SZ", "800000.HK"} {
		w = memory.RecordIndexSkip(w, code)
	}
	if !w.MarketContext.IndicesDone {
		t.Fatal("expected indices_done after all index skips")
	}
}
