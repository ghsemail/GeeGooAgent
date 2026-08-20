package stockdigest_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/stockdigest"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func TestShouldNotifyBlocksFailedStock(t *testing.T) {
	t.Parallel()
	trading := true
	w := memory.NewPreMarketWorking("s1", "postmarket_stock")
	w.IsTradingDay = &trading
	w.Stocks = map[string]memory.StockWorkspace{
		"601766.SH": {Code: "601766.SH", Status: "reported", ReportSummary: "ok"},
		"00700.HK":  {Code: "00700.HK", Status: "failed"},
	}
	result := workflow.RunResult{Working: w, Status: "completed", Supervisor: &workflow.SupervisorReport{Verdict: workflow.VerdictPass}}
	if stockdigest.ShouldNotify("postmarket_stock", "CN", result) {
		t.Fatal("expected no notify when any stock failed")
	}
}

func TestShouldNotifyAllowsSkippedWithExistingReport(t *testing.T) {
	t.Parallel()
	trading := true
	w := memory.NewPreMarketWorking("s1", "postmarket_stock")
	w.IsTradingDay = &trading
	w.Stocks = map[string]memory.StockWorkspace{
		"601766.SH": {
			Code: "601766.SH", Status: "skipped", ReportID: "rid-1",
			ReportSummary: "中国中车今日收涨", ChangePct: 0.8, SessionBias: "bullish",
		},
	}
	result := workflow.RunResult{Working: w, Status: "completed", Supervisor: &workflow.SupervisorReport{Verdict: workflow.VerdictPass}}
	if !stockdigest.ShouldNotify("postmarket_stock", "CN", result) {
		t.Fatal("expected notify for skipped stocks with existing report content")
	}
}

func TestShouldNotifyBlocksEmptyDigest(t *testing.T) {
	t.Parallel()
	trading := true
	w := memory.NewPreMarketWorking("s1", "postmarket_stock")
	w.IsTradingDay = &trading
	w.Stocks = map[string]memory.StockWorkspace{
		"601766.SH": {Code: "601766.SH", Status: "skipped"},
	}
	result := workflow.RunResult{Working: w, Status: "completed", Supervisor: &workflow.SupervisorReport{Verdict: workflow.VerdictPass}}
	if stockdigest.ShouldNotify("postmarket_stock", "CN", result) {
		t.Fatal("expected no notify when no deliverable content")
	}
}
