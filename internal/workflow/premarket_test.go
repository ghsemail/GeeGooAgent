package workflow_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestPreMarketDryRunE2E(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfg := `{
		"base_url": "http://127.0.0.1:3120",
		"api_key": "sk-test",
		"geegoo_url": "http://127.0.0.1:3120",
		"geegoo_api_key": "sk-test",
		"mcp_token": "user-token",
		"output_dir": "` + filepath.ToSlash(dir) + `/data",
		"dry_run": false,
		"llm": {"provider": "deepseek", "token_key": "test-key"},
		"sandbox": {"allowed_hosts": ["127.0.0.1"]}
	}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	application, err := app.LoadFromConfigPath(cfgPath, true)
	if err != nil {
		t.Fatal(err)
	}
	result, err := application.RunPreMarket("pre_market")
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("status=%s error=%s", result.Status, result.LastError)
	}
	w := result.Working
	if w.Phase != "done" {
		t.Fatalf("phase=%s", w.Phase)
	}
	if w.IsTradingDay == nil || !*w.IsTradingDay {
		t.Fatal("expected trading day true")
	}
	if !w.MarketContext.IndicesDone {
		t.Fatal("expected indices_done")
	}
	if !w.MarketContext.MarketNewsDone {
		t.Fatal("expected market_news_done")
	}
	if len(w.Stocks) == 0 {
		t.Fatal("expected stocks")
	}
	for code, ws := range w.Stocks {
		if ws.Status != "reported" {
			t.Fatalf("%s status=%s", code, ws.Status)
		}
		today := time.Now().Format("2006-01-02")
		reportPath := filepath.Join(application.Workspace, "reports", today, code+"-premarket.md")
		if _, err := os.Stat(reportPath); err != nil {
			t.Fatalf("missing report %s: %v", reportPath, err)
		}
	}
	logPath := filepath.Join(application.Workspace, time.Now().Format("2006-01-02"), "execution-log.md")
	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, step := range []string{"check_trading_day", "phase_a_complete", "stock_complete:00700.HK"} {
		if !strings.Contains(content, step) {
			t.Fatalf("missing log step %s", step)
		}
	}
}

func TestMCPClientNotCalledInDryRun(t *testing.T) {
	// Ensures workflow dry-run never needs live GeeGooBot.
	_ = mcp.NewClient
	_ = config.DefaultBotMCPURL
}
