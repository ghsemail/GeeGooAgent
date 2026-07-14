package tools

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
)

func TestHTTPBackendsForTool(t *testing.T) {
	opts := mcp.Options{AllowedHosts: []string{"mcp", "signal", "catalog"}}
	mcpC := mcp.NewClient("http://mcp", "k", opts)
	sigC := mcp.NewClient("http://signal", "k", opts)
	catC := mcp.NewClient("http://catalog", "k", opts)
	b := HTTPBackends{MCP: mcpC, SignalAPI: sigC, SignalCatalog: catC, SignalAnalyze: mcpC}

	if b.ForTool("search_code") != sigC {
		t.Fatal("search_code should use signal-api")
	}
	if b.ForTool("loopback_strategy") != sigC {
		t.Fatal("loopback_strategy should use signal-api")
	}
	if b.ForTool("get_index_signals") != catC {
		t.Fatal("get_index_signals should use catalog-api")
	}
	if b.ForTool("get_signal_combinations") != catC {
		t.Fatal("get_signal_combinations should use catalog-api")
	}
	if b.ForTool("get_position") != mcpC {
		t.Fatal("bot tools should use mcp-api")
	}
}
