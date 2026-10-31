package runtimeapi_test

import (
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
	"github.com/ghsemail/GeeGooAgent/internal/runtimeapi"
	workflowchat "github.com/ghsemail/GeeGooAgent/internal/workflow/chat"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestSignalDiagnoseUIPayloadGet(t *testing.T) {
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
	key := workflowchat.StoreSignalDiagnoseUIPayload(nil, chat, workflowchat.SignalDiagnoseUIEntry{
		RunID: "run-1", StockCode: "00700.HK",
		ChartProbe: map[string]any{"bars": []any{map[string]any{"close": 1.0}}},
		SignalEval: slots.SignalEpisodeEval{Method: "until_opposite_signal"},
	})
	if err := store.Save(chat); err != nil {
		t.Fatal(err)
	}
	application := &app.App{Config: &config.AppConfig{}, DB: db}
	mux := httpserver.NewMux("agent-runtime")
	runtimeapi.NewHandler(application, "").Register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	url := srv.URL + "/v1/signal-diagnose/ui-payload?session_id=" + chat.ID + "&diagnose_ui_key=" + key
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
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
	if out["diagnose_ui_key"] != key {
		t.Fatalf("key=%v", out["diagnose_ui_key"])
	}
	cp, _ := out["chart_probe"].(map[string]any)
	if cp["bars"] == nil {
		t.Fatal("expected bars in fetched payload")
	}
	se, _ := out["signal_eval"].(map[string]any)
	if se["method"] != "until_opposite_signal" {
		t.Fatalf("signal_eval=%v", out["signal_eval"])
	}
}
