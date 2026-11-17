package runtimeapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/httpserver"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/runtimeapi"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
	workflowchat "github.com/ghsemail/GeeGooAgent/internal/workflow/chat"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

// POST /v1/signal-diagnose/lab-detail with daily_macro must inject MCP trading-day checker and fail-closed when MCP rejects.
func TestSignalDiagnoseLabDetailHTTP_DailyMacroMCPFailClosed(t *testing.T) {
	mcpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 100, "message": "success",
			"data": map[string]any{"is_trading_day": false, "date": "2026-11-16", "market": "CN", "code": "000300.SH"},
		})
	}))
	defer mcpSrv.Close()

	labSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{"bars": []any{map[string]any{"time": "2026-11-16 00:00:00", "close": 200.0}}},
		})
	}))
	defer labSrv.Close()

	root := t.TempDir()
	db, err := infra.OpenSQLite(filepath.Join(root, "geegoo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	store := chatsession.NewSQLiteSessionStore(db)
	chat, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}
	runID := "flow-macro-http"
	ref := strategydev.LabDetailRef("buy", 0)
	chart := []any{map[string]any{"time": "2026-11-03 10:00:00", "close": 99.0}}
	flow := workflowchat.Flow{
		RunID: runID, Template: workflowchat.SkillSignalDiagnose, LabMode: true, StockCode: "000300.SH",
		ProbeRaw: map[string]any{"bars": chart},
	}
	chat.Metadata = map[string]any{
		workflowchat.MetaKeyLabDiagnoseSnapshots: map[string]json.RawMessage{
			runID: workflowchat.MustLabDiagnoseSnapshotJSON(flow),
		},
	}
	if err := store.Save(chat); err != nil {
		t.Fatal(err)
	}

	cfg := &config.AppConfig{LabAPIURLField: labSrv.URL, LabAPIKeyField: "lab-key"}
	application := &app.App{
		Config:   cfg,
		DB:       db,
		Registry: tools.NewRegistry(),
		MCP: mcp.NewClient(mcpSrv.URL, "sk-test", mcp.Options{
			AllowedHosts: []string{"127.0.0.1", "localhost"},
			MaxRetries:   1,
			RetryWait:    0,
			Sleep:        func(time.Duration) {},
		}),
	}

	mux := httpserver.NewMux("agent-runtime")
	runtimeapi.NewHandler(application, "").Register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	body, _ := json.Marshal(map[string]any{
		"session_id": chat.ID, "run_id": runID, "detail_ref": ref,
		"kline_jev_profile": strategydev.LabKlineJEVProfileDailyMacro,
		"mcp_token":         "user-mcp-token",
	})
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+"/v1/signal-diagnose/lab-detail", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-MCP-Token", "user-mcp-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	detail, _ := out["detail"].(map[string]any)
	degraded, _ := detail["degraded"].([]any)
	hasFail := false
	for _, d := range degraded {
		if s, _ := d.(string); s == "macro_trading_day_check_failed" || s == "macro_monitor_cutoff_unavailable" {
			hasFail = true
		}
	}
	if !hasFail {
		t.Fatalf("expected trading-day fail-closed degraded, detail=%v", detail)
	}
	if detail["price_analysis"] != nil {
		t.Fatalf("expected no price_analysis on fail-closed, got %v", detail["price_analysis"])
	}
}