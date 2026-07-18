package runtimeapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/auth"
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/httpserver"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/runtimeapi"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func testChatStreamApp(t *testing.T) *app.App {
	t.Helper()
	registry := tools.NewRegistry()
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{Content: "SSE 回复内容。", Usage: llm.TokenUsage{Model: "mock"}},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	state := infra.NewStateStore(t.TempDir())
	return &app.App{
		Config:   &config.AppConfig{},
		State:    state,
		Registry: registry,
		Gateway:  gateway,
		Agent:    agent.New(gateway, runtime.NewExecutor(registry), registry),
	}
}

func TestChatStreamTurn(t *testing.T) {
	application := testChatStreamApp(t)
	mux := httpserver.NewMux("agent-runtime")
	runtimeapi.NewHandler(application).Register(mux)
	handler := auth.SkipPaths(map[string]struct{}{"/health": {}}, auth.BearerAPIKey("test-runtime-key"))(mux)

	payload := map[string]string{"message": "你好"}
	raw, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/stream", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{"event: connected", "event: turn_end", "event: done", "SSE 回复内容。"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in body: %s", want, body)
		}
	}
}

func TestSessionEventsStreamProgress(t *testing.T) {
	application := testChatStreamApp(t)
	store, err := application.SessionStore()
	if err != nil {
		t.Fatal(err)
	}
	session, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}
	pub := chatsession.NewLivePublisher(application.State, session.ID)
	pub.Emit("turn_start", map[string]any{"message": "hi"})
	pub.Emit("stream_delta", map[string]any{"content": "chunk"})

	mux := httpserver.NewMux("agent-runtime")
	runtimeapi.NewHandler(application).Register(mux)
	handler := auth.SkipPaths(map[string]struct{}{"/health": {}}, auth.BearerAPIKey("test-runtime-key"))(mux)

	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/events/stream?session_id="+session.ID+"&interval_ms=100", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: connected") {
		t.Fatalf("missing connected: %s", body)
	}
	if !strings.Contains(body, "event: progress") {
		t.Fatalf("missing progress events: %s", body)
	}
	if !strings.Contains(body, `"event":"stream_delta"`) {
		t.Fatalf("missing stream_delta progress: %s", body)
	}
}

func TestChatStreamRequiresMessage(t *testing.T) {
	application := testChatStreamApp(t)
	handler := testProtectedHandler(t, application)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/stream", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
