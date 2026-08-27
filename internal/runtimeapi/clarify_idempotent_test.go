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
	var resolved PendingClarify
	var resolvedAnswer string
	var auto bool
	answer, ok := handler.waitClarifyOrAuto(ctx, "sess-auto", "回测频率用哪个？", []string{
		"daily（日线，长期）",
		"60m（小时，中短线）",
	}, ClarifyHooks{
		OnResolved: func(p PendingClarify, ans string, pickedAuto bool) {
			resolved = p
			resolvedAnswer = ans
			auto = pickedAuto
		},
	})
	if !ok || answer != "60m（小时，中短线）" {
		t.Fatalf("answer=%q ok=%v", answer, ok)
	}
	if resolved.SessionID != "sess-auto" || resolvedAnswer != answer || !auto {
		t.Fatalf("resolved=%+v answer=%q auto=%v", resolved, resolvedAnswer, auto)
	}
}

func TestWaitClarifyOrAutoEmitsResolvedHookForManualAnswer(t *testing.T) {
	h := newClarifyHub()
	handler := &Handler{clarify: h}
	done := make(chan struct{})
	go func() {
		ans, ok := handler.waitClarifyOrAuto(context.Background(), "sess-manual", "pick one", []string{"A", "B"}, ClarifyHooks{
			OnResolved: func(p PendingClarify, answer string, pickedAuto bool) {
				if !pickedAuto && answer == "B" {
					close(done)
				}
			},
		})
		if !ok || ans != "B" {
			t.Errorf("wait=%q ok=%v", ans, ok)
		}
	}()
	time.Sleep(10 * time.Millisecond)
	if !h.Answer("sess-manual", "B", true) {
		t.Fatal("answer failed")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting resolved hook")
	}
}
