package runtimeapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/httpserver"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/runtimeapi"
	workflowchat "github.com/ghsemail/GeeGooAgent/internal/workflow/chat"
)

func TestSignalDiagnoseLabDetailPersistResultWithoutArchiveSnapshot(t *testing.T) {
	t.Parallel()
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
	rt := &runtime.Session{ID: chat.ID}
	workflowchat.UpdateSignalDiagnoseResultAfterTool(rt, chat, "probe_bot_signal_series", map[string]any{
		"code": "00700.HK", "frequency": "60m",
		"bars": []any{map[string]any{"close": 1.0, "time": "t0"}},
		"buy_merged": []any{1}, "sell_merged": []any{0},
	})
	delete(chat.Metadata, workflowchat.MetaKeyLabDiagnoseSnapshots)
	if err := store.Save(chat); err != nil {
		t.Fatal(err)
	}

	application := &app.App{Config: &config.AppConfig{}, DB: db}
	mux := httpserver.NewMux("agent-runtime")
	runtimeapi.NewHandler(application, "").Register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mondayRun := "6ae5e1ca7e27c6bcad2e4384"
	body, _ := json.Marshal(map[string]any{
		"session_id": chat.ID, "run_id": mondayRun, "detail_ref": "not-a-ref",
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
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 bad ref when flow resolved from persist result, got %d", resp.StatusCode)
	}
}
