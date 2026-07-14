package workflow_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func TestSupervisorPassOnDoneWithReportedStocks(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	eng := workflow.NewEngine(dir, workflow.DefaultPreMarketChecks())
	trading := true
	code := "00700.HK"
	w := memory.NewPreMarketWorking("s1", "pre_market")
	w.Phase = "done"
	w.IsTradingDay = &trading
	w.Stocks[code] = memory.StockWorkspace{
		Code: code, Status: "reported", ReportID: "r1",
		BotID: "b1", BotName: "bot", BotType: "DCA", ReportRef: filepath.Join(dir, "reports", "2026-07-04", code+"-premarket.md"),
	}
	// Create the local md so file_exists passes.
	mdDir := filepath.Join(dir, "reports", "2026-07-04")
	if err := os.MkdirAll(mdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mdDir, code+"-premarket.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	report := eng.Verify(w, "2026-07-04")
	if report.Verdict != workflow.VerdictPass {
		t.Fatalf("expected pass, got %s: %s", report.Verdict, report.Summary())
	}
}

func TestSupervisorRecoverableOnMissingReportID(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	eng := workflow.NewEngine(dir, workflow.DefaultPreMarketChecks())
	trading := true
	w := memory.NewPreMarketWorking("s2", "pre_market")
	w.Phase = "done"
	w.IsTradingDay = &trading
	w.Stocks["00700.HK"] = memory.StockWorkspace{
		Code: "00700.HK", Status: "reported", // missing report_id
		BotID: "b1", BotName: "bot", BotType: "DCA",
	}
	report := eng.Verify(w, time.Now().Format("2006-01-02"))
	if report.Verdict != workflow.VerdictRecoverable {
		t.Fatalf("expected recoverable, got %s", report.Verdict)
	}
	if len(report.MissingSteps) == 0 {
		t.Fatal("expected missing steps listed")
	}
}

func TestSupervisorTerminalOnFailedStock(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	eng := workflow.NewEngine(dir, workflow.DefaultPreMarketChecks())
	trading := true
	w := memory.NewPreMarketWorking("s3", "pre_market")
	w.Phase = "phase_b"
	w.IsTradingDay = &trading
	w.Stocks["00700.HK"] = memory.StockWorkspace{Code: "00700.HK", Status: "failed"}
	report := eng.Verify(w, "2026-07-04")
	if report.Verdict != workflow.VerdictTerminal {
		t.Fatalf("expected terminal, got %s", report.Verdict)
	}
}

func TestSupervisorPassOnNonTradingDay(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	eng := workflow.NewEngine(dir, workflow.DefaultPreMarketChecks())
	trading := false
	w := memory.NewPreMarketWorking("s4", "pre_market")
	w.Phase = "done"
	w.IsTradingDay = &trading
	report := eng.Verify(w, "2026-07-04")
	if report.Verdict != workflow.VerdictPass {
		t.Fatalf("non-trading day should pass, got %s: %s", report.Verdict, report.Summary())
	}
}

func TestClassifyError(t *testing.T) {
	t.Parallel()
	if workflow.ClassifyForTest("get_mcp_analysis", "context deadline exceeded (timeout)") != workflow.ErrorRecoverable {
		t.Fatal("timeout should be recoverable")
	}
	if workflow.ClassifyForTest("create_pre_market_report", "401 unauthorized: missing mcp_token") != workflow.ErrorTerminal {
		t.Fatal("401 should be terminal")
	}
}
