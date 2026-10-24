package runtimeapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/httpserver"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/runtimeapi"
	workflowchat "github.com/ghsemail/GeeGooAgent/internal/workflow/chat"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func TestSignalDiagnoseLabDetailCacheHitSkipsJEV(t *testing.T) {
	t.Parallel()
	var jevCalls atomic.Int32
	jevSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		jevCalls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer jevSrv.Close()

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
	runID := "flow-20261024T130000"
	ref := strategydev.LabDetailRef("buy", 0)
	flow := workflowchat.Flow{
		RunID: runID, Template: workflowchat.SkillSignalDiagnose, LabMode: true,
		ProbeRaw: map[string]any{"bars": []any{map[string]any{"close": 10.0, "time": float64(1)}}},
	}
	rawFlow := workflowchat.MustLabDiagnoseSnapshotJSON(flow)
	chat.Metadata = map[string]any{
		workflowchat.MetaKeyLabDiagnoseSnapshots: map[string]json.RawMessage{runID: rawFlow},
	}
	workflowchat.StoreLabEpisodeDetail(chat, nil, runID, ref, strategydev.LabEpisodeDetail{
		Ref: ref, Side: "buy", BarIndex: 0, Close: 99,
		PriceAnalysisJEV: &strategydev.PriceAnalysisJEV{OK: true, RecommendedChoice: "cached"},
	})
	if err := store.Save(chat); err != nil {
		t.Fatal(err)
	}

	application := &app.App{Config: &config.AppConfig{}, DB: db}
	mux := httpserver.NewMux("agent-runtime")
	runtimeapi.NewHandler(application, "").Register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	body, _ := json.Marshal(map[string]any{
		"session_id": chat.ID, "run_id": runID, "detail_ref": ref,
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
	if out["cached"] != true {
		t.Fatalf("expected cached=true, got %v", out["cached"])
	}
	detail, _ := out["detail"].(map[string]any)
	if detail["close"] != 99.0 {
		t.Fatalf("detail=%v", detail)
	}
	if jevCalls.Load() != 0 {
		t.Fatalf("expected no JEV calls on cache hit, got %d", jevCalls.Load())
	}
}

func TestSignalDiagnoseLabDetailInvalidRef(t *testing.T) {
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
	runID := "flow-20261024T140000"
	flow := workflowchat.Flow{
		RunID: runID, Template: workflowchat.SkillSignalDiagnose, LabMode: true,
		ProbeRaw: map[string]any{"bars": []any{map[string]any{"close": 1.0}}},
	}
	chat.Metadata = map[string]any{
		workflowchat.MetaKeyLabDiagnoseSnapshots: map[string]json.RawMessage{
			runID: workflowchat.MustLabDiagnoseSnapshotJSON(flow),
		},
	}
	if err := store.Save(chat); err != nil {
		t.Fatal(err)
	}
	application := &app.App{Config: &config.AppConfig{}, DB: db}
	mux := httpserver.NewMux("agent-runtime")
	runtimeapi.NewHandler(application, "").Register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	body, _ := json.Marshal(map[string]any{
		"session_id": chat.ID, "run_id": runID, "detail_ref": "not-a-ref",
	})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+"/v1/signal-diagnose/lab-detail", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-MCP-Token", "user-mcp-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
