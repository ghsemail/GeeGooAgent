package strategydev

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJevClientDecisionChat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/decision/chat" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer data-token") {
			t.Fatalf("auth=%q", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "message": "ok",
			"data": map[string]any{
				"model":   "jev-1.13.0",
				"content": `{"buy_signal_id":"b1","sell_signal_id":"s1","reason":"ok"}`,
			},
		})
	}))
	defer srv.Close()

	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "data-token", Model: "jev-1.13.0"})
	buy, sell, reason, err := jev.PickStrategies(context.Background(), RegimeUp, "long", "60m",
		[]Candidate{{SignalID: "b1", Name: "B"}},
		[]Candidate{{SignalID: "s1", Name: "S"}},
	)
	if err != nil || buy != "b1" || sell != "s1" || reason != "ok" {
		t.Fatalf("pick: buy=%q sell=%q reason=%q err=%v", buy, sell, reason, err)
	}
}

func TestJevClientDecisionChoiceAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 4001, "message": "upstream failed",
		})
	}))
	defer srv.Close()

	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "k", Model: "jev-1.13.0"})
	_, _, _, _, err := jev.Choice(context.Background(), "state", "direction", "pick one", map[string]string{"看多": "up"})
	if err == nil || !strings.Contains(err.Error(), "upstream failed") {
		t.Fatalf("err=%v", err)
	}
}
