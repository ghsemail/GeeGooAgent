package workflow_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/app"
)

func TestIntradayDryRunE2E(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfg := `{
		"base_url": "http://127.0.0.1:3120",
		"api_key": "sk-test",
		"geegoo_url": "http://127.0.0.1:3120",
		"geegoo_api_key": "sk-test",
		"mcp_token": "user-token",
		"output_dir": "` + filepath.ToSlash(dir) + `/data",
		"dry_run": true,
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
	defer application.Close()
	result, err := application.RunSkill("intraday")
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("status=%s err=%s", result.Status, result.LastError)
	}
	if len(result.Working.Stocks) != 1 {
		t.Fatalf("expected 1 stock, got %d", len(result.Working.Stocks))
	}
}

func TestPostMarketDryRunE2E(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfg := `{
		"base_url": "http://127.0.0.1:3120",
		"api_key": "sk-test",
		"geegoo_url": "http://127.0.0.1:3120",
		"geegoo_api_key": "sk-test",
		"mcp_token": "user-token",
		"output_dir": "` + filepath.ToSlash(dir) + `/data",
		"dry_run": true,
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
	defer application.Close()
	result, err := application.RunSkill("post_market")
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("status=%s err=%s", result.Status, result.LastError)
	}
	if result.Working.Phase != "done" {
		t.Fatalf("phase=%s", result.Working.Phase)
	}
}
