package mcp_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
)

func TestShouldAnalyzeFallback(t *testing.T) {
	t.Parallel()
	code := 101
	if mcp.ShouldAnalyzeFallback(&mcp.ClientError{Message: "缺少 signal_id", APICode: &code}) {
		t.Fatal("business code 101 should not fallback")
	}
	if !mcp.ShouldAnalyzeFallback(&mcp.ClientError{Message: "server error 502 for /x", HTTPStatus: 502}) {
		t.Fatal("HTTP 502 should fallback")
	}
	if !mcp.ShouldAnalyzeFallback(&mcp.ClientError{Message: "transport error: dial tcp"}) {
		t.Fatal("transport error should fallback")
	}
}
