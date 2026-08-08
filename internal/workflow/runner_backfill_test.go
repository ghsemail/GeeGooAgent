package workflow_test

import (
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func TestShouldRunPerStockPhaseOnBackfillNonTradingDay(t *testing.T) {
	trading := false
	w := memory.NewPreMarketWorking("s1", "postmarket_stock")
	w.IsTradingDay = &trading
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	workflow.SeedReportDate(w, yesterday)
	if !workflow.IsBackfillRun(w) {
		t.Fatal("expected backfill when report_date is yesterday")
	}
	if !workflow.ShouldRunPerStockPhaseForTest(w) {
		t.Fatal("per-stock phase should run on backfill even when today is not a trading day")
	}
}

func TestShouldNotRunPerStockPhaseWhenNotTradingDayAndNotBackfill(t *testing.T) {
	trading := false
	w := memory.NewPreMarketWorking("s1", "postmarket_stock")
	w.IsTradingDay = &trading
	if workflow.ShouldRunPerStockPhaseForTest(w) {
		t.Fatal("per-stock phase should be skipped on non-trading day without backfill")
	}
}
