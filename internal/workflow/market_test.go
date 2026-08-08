package workflow_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func TestBuildMarketReportContentCN(t *testing.T) {
	t.Parallel()
	w := memory.NewPreMarketWorking("s1", "premarket_market")
	w.Market = "CN"
	w.MarketContext.IndexAnalysisRefs = map[string]string{
		"000001.SH": "上证偏强，量能温和放大",
		"399001.SZ": "深成指震荡整理",
	}
	w.MarketContext.MarketNews = map[string]string{"CN": "- 政策面偏暖\n- 北向资金净流入"}
	body := workflow.BuildMarketReportContent(w, "CN")
	for _, want := range []string{"指数概览", "000001.SH", "399001.SZ", "市场新闻解读", "政策面偏暖", "GeeGoo 智能体市场盘前 skill"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in:\n%s", want, body)
		}
	}
	if strings.Contains(body, "道琼斯") || strings.Contains(body, "恒生") {
		t.Fatalf("CN report should not mention other markets:\n%s", body)
	}
}

func TestSupervisorMarketPreMarketPass(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	date := "2026-08-08"
	market := "CN"
	eng := workflow.NewEngine(dir, workflow.DefaultMarketPreMarketChecks())
	w := memory.NewPreMarketWorking("s1", "premarket_market")
	w.Phase = "done"
	w.Market = market
	w.MarketReportID = "mkt-1"
	w.MarketContext.IndicesDone = true
	w.MarketContext.MarketNewsDone = true
	mdDir := filepath.Join(dir, "reports", date)
	if err := os.MkdirAll(mdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mdPath := filepath.Join(mdDir, "market-"+market+"-market_premarket.md")
	if err := os.WriteFile(mdPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	report := eng.Verify(w, date)
	if report.Verdict != workflow.VerdictPass {
		t.Fatalf("expected pass, got %s: %s", report.Verdict, report.Summary())
	}
}

func TestSupervisorMarketPreMarketRecoverableMissingReportID(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	eng := workflow.NewEngine(dir, workflow.DefaultMarketPreMarketChecks())
	w := memory.NewPreMarketWorking("s2", "premarket_market")
	w.Phase = "done"
	w.Market = "HK"
	w.MarketContext.IndicesDone = true
	w.MarketContext.MarketNewsDone = true
	report := eng.Verify(w, time.Now().Format("2006-01-02"))
	if report.Verdict != workflow.VerdictRecoverable {
		t.Fatalf("expected recoverable, got %s", report.Verdict)
	}
}
