package runtimeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/app"
)

func TestChatClarifyAlreadyResolvedIsIdempotent(t *testing.T) {
	h := NewHandler(&app.App{}, "")
	body, _ := json.Marshal(map[string]any{
		"session_id": "sess-idempotent",
		"answer":     "60m",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/clarify", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.chatClarify(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["already_resolved"] != true {
		t.Fatalf("resp=%v", resp)
	}
}

func TestWaitClarifyOrAutoEmitsAutoResolvedHook(t *testing.T) {
	h := newClarifyHub()
	handler := &Handler{clarify: h}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	var autoEvent PendingClarify
	var autoAnswer string
	answer, ok := handler.waitClarifyOrAuto(ctx, "sess-auto", "回测频率用哪个？", []string{
		"daily（日线，长期）",
		"60m（小时，中短线）",
	}, ClarifyHooks{
		OnAutoResolved: func(p PendingClarify, ans string) {
			autoEvent = p
			autoAnswer = ans
		},
	})
	if !ok || answer != "60m（小时，中短线）" {
		t.Fatalf("answer=%q ok=%v", answer, ok)
	}
	if autoEvent.SessionID != "sess-auto" || autoAnswer != answer {
		t.Fatalf("autoEvent=%+v autoAnswer=%q", autoEvent, autoAnswer)
	}
}
