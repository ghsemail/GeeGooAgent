package runtimeapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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

// Injectable cross-service contract test: MCP fixture must echo queried calendar day (not legacy today).
func TestSignalDiagnoseLabDetailHTTP_DailyMacroLegacyDateEchoFailClosed(t *testing.T) {
	fixturePath := filepath.Join("..", "clients", "mcp", "testdata", "trading_day", "legacy_today_mismatch.json")
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	var td map[string]any
	if err := json.Unmarshal(raw, &td); err != nil {
		t.Fatal(err)
	}
	mcpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 100, "message": "success", "data": td,
		})
	}))
	defer mcpSrv.Close()

	labSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{"bars": []any{map[string]any{"time": "2020-07-01 00:00:00", "close": 200.0}}},
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
	runID := "flow-macro-cal"
	ref := strategydev.LabDetailRef("buy", 0)
	chart := []any{map[string]any{"time": "2020-07-01 10:00:00", "close": 99.0}}
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
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+"/v1/signal-diagnose/lab-detail", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-MCP-Token", "user-mcp-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	detail, _ := out["detail"].(map[string]any)
	if detail["price_analysis"] != nil {
		t.Fatalf("legacy echo must fail-closed, got PA %v", detail["price_analysis"])
	}
}