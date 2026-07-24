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

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/httpserver"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/runtimeapi"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestPlanHTTPApproveResumesPersistedPlan(t *testing.T) {
	t.Parallel()
	called := false
	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name: "create_dca_bot",
		Handle: tools.ApprovalGate("create_dca_bot", func(ctx tools.Context, args map[string]any) tools.Result {
			called = true
			return tools.Result{Status: tools.StatusOK, Summary: "created"}
		}),
	})
	provider := &llm.MockProvider{
		Responses: []*llm.Response{{Content: "已完�?}},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})

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
	chat.SyncHeldPlan(1, []llm.ToolCall{{
		ID: "c1", Name: "create_dca_bot", Arguments: map[string]any{"botname": "t"},
	}})
	if err := store.Save(chat); err != nil {
		t.Fatal(err)
	}

	application := &app.App{
		Config:   &config.AppConfig{},
		Registry: registry,
		Gateway:  gateway,
		Agent:    agent.New(gateway, runtime.NewExecutor(registry), registry),
		DB:       db,
	}
	application.Agent.SetPlanGate(true)
	mux := httpserver.NewMux("agent-runtime")
	runtimeapi.NewHandler(application, "").Register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	body, _ := json.Marshal(map[string]any{
		"session_id": chat.ID,
		"approve":    true,
	})
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+"/v1/chat/plan", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var out struct {
		PlanPending   bool   `json:"plan_pending"`
		AssistantText string `json:"assistant_text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.PlanPending {
		t.Fatalf("still pending: %+v", out)
	}
	if !called {
		t.Fatal("mutating tool should run after approve")
	}
	loaded, err := store.Load(chat.ID)
	if err != nil || loaded == nil {
		t.Fatal(err)
	}
	if _, _, ok := loaded.HeldPlanFromMetadata(); ok {
		t.Fatal("pending plan should be cleared in storage")
	}
}
