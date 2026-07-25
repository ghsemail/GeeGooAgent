package runtimeapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveChatMCPTokenPrefersConfig(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/stream", nil)
	req.Header.Set("X-MCP-Token", "admin-token")
	got := resolveChatMCPToken(req, "body-token", "config-trading-token")
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
