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
	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestSignalDiagnoseResultPostLatest(t *testing.T) {
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
		"code": "00700.HK", "bars": []any{map[string]any{"close": 1.0}}, "buy_merged": []any{1}, "sell_merged": []any{0},
	})
	if err := store.Save(chat); err != nil {
		t.Fatal(err)
	}
	application := &app.App{Config: &config.AppConfig{}, DB: db}
	mux := httpserver.NewMux("agent-runtime")
	runtimeapi.NewHandler(application, "").Register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	body, _ := json.Marshal(map[string]string{"session_id": chat.ID})
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+"/v1/signal-diagnose/result", bytes.NewReader(body))
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
	cp, _ := out["chart_probe"].(map[string]any)
	if cp["bars"] == nil {
		t.Fatal("expected chart_probe.bars")
	}
	se, _ := out["signal_eval"].(map[string]any)
	if se["method"] == nil {
		t.Fatalf("signal_eval=%v", out["signal_eval"])
	}
	if out["trade_direction"] != "both" {
		t.Fatalf("trade_direction=%v want both", out["trade_direction"])
	}
}

func TestSignalDiagnoseResultPostByRunID(t *testing.T) {
	t.Parallel()
	chat := &chatsession.ChatSession{ID: "sess-1", Metadata: map[string]any{}}
	rt := runtime.NewSession()
	rt.ID = chat.ID
	flow := &workflowchat.Flow{
		Template: workflowchat.SkillSignalDiagnose, RunID: "run-monday-1", StockCode: "00700.HK",
		Status: workflowchat.StatusCompleted, EvalJudgment: "judgment",
		ProbeRaw: map[string]any{
			"bars": []any{map[string]any{"close": 2.0}}, "buy_merged": []any{1}, "sell_merged": []any{0},
		},
		SignalEval: slots.SignalEpisodeEval{Method: "until_opposite_signal"},
	}
	workflowchat.SaveToSession(rt, flow)
	workflowchat.FinalizeSignalDiagnoseResultFromSession(rt, chat)
	ent, ok := workflowchat.LookupSignalDiagnoseResultForMonday(chat, "run-monday-1")
	if !ok || ent == nil || ent.RunID != "run-monday-1" {
		t.Fatalf("lookup by run_id failed ok=%v ent=%+v", ok, ent)
	}
	_, okLatest := workflowchat.LookupSignalDiagnoseResultForMonday(chat, "")
	if !okLatest {
		t.Fatal("expected latest when run_id omitted")
	}
}

func TestSignalDiagnoseResultPostIncludesRunIDAndTradeDirection(t *testing.T) {
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
	rt := runtime.NewSession()
	rt.ID = chat.ID
	flow := &workflowchat.Flow{
		Template: workflowchat.SkillSignalDiagnose, RunID: "flow-test-1", StockCode: "00700.HK",
		TradeDirection: runtime.TradeDirectionLongOnly,
		Status: workflowchat.StatusCompleted,
		ProbeRaw: map[string]any{
			"bars": []any{map[string]any{"close": 2.0}}, "buy_merged": []any{1}, "sell_merged": []any{0},
		},
		SignalEval: slots.SignalEpisodeEval{Method: "until_opposite_signal"},
	}
	workflowchat.SaveToSession(rt, flow)
	workflowchat.FinalizeSignalDiagnoseResultFromSession(rt, chat)
	if err := store.Save(chat); err != nil {
		t.Fatal(err)
	}
	application := &app.App{Config: &config.AppConfig{}, DB: db}
	mux := httpserver.NewMux("agent-runtime")
	runtimeapi.NewHandler(application, "").Register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	body, _ := json.Marshal(map[string]string{"session_id": chat.ID})
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+"/v1/signal-diagnose/result", bytes.NewReader(body))
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
	if out["run_id"] != "flow-test-1" {
		t.Fatalf("run_id=%v", out["run_id"])
	}
	if out["trade_direction"] != "long_only" {
		t.Fatalf("trade_direction=%v", out["trade_direction"])
	}
}
