package runtimeapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/auth"
	"github.com/ghsemail/GeeGooAgent/internal/httpserver"
	"github.com/ghsemail/GeeGooAgent/internal/runtimeapi"
)

func TestSkillsRunIntradayDryRun(t *testing.T) {
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

	mux := httpserver.NewMux("agent-runtime")
	runtimeapi.NewHandler(application, cfgPath).Register(mux)
	handler := auth.SkipPaths(map[string]struct{}{"/health": {}}, auth.BearerAPIKey("test-runtime-key"))(mux)

	body := map[string]any{
		"skill": "intraday",
		"intraday": map[string]string{
			"code": "SPCX.US", "stock_name": "SpaceX", "bot_id": "bot-1",
			"bot_type": "DCA", "frequency": "60m", "trade_type": "信号买入",
		},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/skills/run", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	dec, _ := resp["decision"].(map[string]any)
	if dec == nil {
		t.Fatalf("missing decision: %v", resp)
	}
	if dec["result"] == nil || dec["confidence"] == nil {
		t.Fatalf("decision=%v", dec)
	}
}
