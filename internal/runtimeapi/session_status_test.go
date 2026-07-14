package runtimeapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/auth"
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/httpserver"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtimeapi"
)

func testSessionApp(t *testing.T) *app.App {
	t.Helper()
	state := infra.NewStateStore(t.TempDir())
	store := chatsession.NewChatSessionStore(state)
	session, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}
	session.Title = "debug session"
	session.Messages = append(session.Messages, llm.Message{Role: llm.RoleUser, Content: "hello"})
	session.StepRecords = append(session.StepRecords, chatsession.ChatStepRecord{
		Step: 1, Kind: "reply", Summary: "hi", Timestamp: time.Now().UTC(),
	})
	if err := store.Save(session); err != nil {
		t.Fatal(err)
	}
	pub := chatsession.NewLivePublisher(state, session.ID)
	pub.Emit("turn_start", nil)
	return &app.App{Config: &config.AppConfig{}, State: state}
}

func testProtectedHandler(t *testing.T, application *app.App) http.Handler {
	t.Helper()
	mux := httpserver.NewMux("agent-runtime")
	runtimeapi.NewHandler(application).Register(mux)
	return auth.SkipPaths(map[string]struct{}{"/health": {}}, auth.BearerAPIKey("test-runtime-key"))(mux)
}

func TestSessionStatusJSONLatest(t *testing.T) {
	application := testSessionApp(t)
	handler := testProtectedHandler(t, application)

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/status", nil)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload runtimeapi.SessionStatusPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.SessionID == "" {
		t.Fatal("empty session_id")
	}
	if payload.ResolvedFrom != "latest" {
		t.Fatalf("resolved_from=%q", payload.ResolvedFrom)
	}
	if payload.MessageCount < 2 {
		t.Fatalf("message_count=%d", payload.MessageCount)
	}
	if !payload.Busy {
		t.Fatal("expected busy live state")
	}
}

func TestSessionStatusStreamSnapshot(t *testing.T) {
	application := testSessionApp(t)
	handler := testProtectedHandler(t, application)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/status/stream?interval_ms=200", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: connected") {
		t.Fatalf("missing connected event: %s", body)
	}
	if !strings.Contains(body, "event: snapshot") {
		t.Fatalf("missing snapshot event: %s", body)
	}
	if !strings.Contains(body, `"session_id"`) {
		t.Fatalf("missing session payload: %s", body)
	}
}

func TestSessionStatusNotFound(t *testing.T) {
	application := &app.App{Config: &config.AppConfig{}, State: infra.NewStateStore(t.TempDir())}
	handler := testProtectedHandler(t, application)

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/status", nil)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
