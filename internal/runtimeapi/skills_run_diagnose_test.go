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

func testAppForSkillsRun(t *testing.T) (*app.App, string) {
	t.Helper()
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
	return application, cfgPath
}

func TestSkillsRunDiagnoseStrategyValidation(t *testing.T) {
	application, cfgPath := testAppForSkillsRun(t)
	defer application.Close()
	mux := httpserver.NewMux("agent-runtime")
	runtimeapi.NewHandler(application, cfgPath).Register(mux)
	handler := auth.SkipPaths(map[string]struct{}{"/health": {}}, auth.BearerAPIKey("test-runtime-key"))(mux)

	body := map[string]any{"skill": "diagnose_strategy", "strategy": map[string]string{"code": "00700.HK"}}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/skills/run", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSkillsRunDiagnoseMacroPoolValidation(t *testing.T) {
	application, cfgPath := testAppForSkillsRun(t)
	defer application.Close()
	mux := httpserver.NewMux("agent-runtime")
	runtimeapi.NewHandler(application, cfgPath).Register(mux)
	handler := auth.SkipPaths(map[string]struct{}{"/health": {}}, auth.BearerAPIKey("test-runtime-key"))(mux)

	body := map[string]any{"skill": "diagnose_macro_pool", "macro_pool": map[string]any{"symbols": []any{}}}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/skills/run", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSkillsRunDiagnoseMacroOverallValidation(t *testing.T) {
	application, cfgPath := testAppForSkillsRun(t)
	defer application.Close()
	mux := httpserver.NewMux("agent-runtime")
	runtimeapi.NewHandler(application, cfgPath).Register(mux)
	handler := auth.SkipPaths(map[string]struct{}{"/health": {}}, auth.BearerAPIKey("test-runtime-key"))(mux)

	body := map[string]any{
		"skill": "diagnose_macro_overall",
		"macro_overall": map[string]any{"date": "2026-11-22", "pools": []any{}},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/skills/run", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSkillsRunDiagnoseMacroValidation(t *testing.T) {
	application, cfgPath := testAppForSkillsRun(t)
	defer application.Close()
	mux := httpserver.NewMux("agent-runtime")
	runtimeapi.NewHandler(application, cfgPath).Register(mux)
	handler := auth.SkipPaths(map[string]struct{}{"/health": {}}, auth.BearerAPIKey("test-runtime-key"))(mux)

	body := map[string]any{"skill": "diagnose_macro", "macro": map[string]string{}}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/skills/run", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}