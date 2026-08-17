package runtimeapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveChatMCPTokenPrefersCaller(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/stream", nil)
	req.Header.Set("X-MCP-Token", "user-token")
	got := resolveChatMCPToken(req, "body-token", "config-trading-token")
	if got != "user-token" {
		t.Fatalf("got %q want user-token", got)
	}
}

func TestResolveChatMCPTokenFallsBackToConfig(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/stream", nil)
	got := resolveChatMCPToken(req, "", "config-trading-token")
	if got != "config-trading-token" {
		t.Fatalf("got %q want config-trading-token", got)
	}
}

func TestResolveChatMCPTokenFallsBackToCaller(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/stream", nil)
	req.Header.Set("X-MCP-Token", "caller-token")
	got := resolveChatMCPToken(req, "", "")
	if got != "caller-token" {
		t.Fatalf("got %q want caller-token", got)
	}
}
