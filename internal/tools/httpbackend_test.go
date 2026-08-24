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
	if b.ForTool("probe_bot_signal_series") != sigC {
		t.Fatal("probe_bot_signal_series should use signal-api")
	}
	if b.ForTool("list_strategy_backtest_logs") != sigC {
		t.Fatal("list_strategy_backtest_logs should use signal-api")
	}
	if b.ForTool("run_strategy_backtest") != sigC {
		t.Fatal("run_strategy_backtest should use signal-api")
	}
	if b.ForTool("get_index_signals") != mcpC {
		t.Fatal("get_index_signals should use mcp-api")
	}
	if b.ForTool("get_signal_combinations") != mcpC {
		t.Fatal("get_signal_combinations should use mcp-api")
	}
	if b.ForTool("add_index_signal") != mcpC {
		t.Fatal("add_index_signal should use mcp-api")
	}
	analyzeC := mcp.NewClient("http://analyze", "k", opts)
	b2 := HTTPBackends{MCP: mcpC, SignalAPI: sigC, SignalCatalog: catC, SignalAnalyze: analyzeC}
	if b2.ForTool("generate_grid_strategy") != analyzeC {
		t.Fatal("generate_grid_strategy should use analyze-api")
	}
	if b2.ForTool("generate_dca_strategy") != analyzeC {
		t.Fatal("generate_dca_strategy should use analyze-api")
	}
	if b2.ForTool("get_mcp_analysis") != mcpC {
		t.Fatal("get_mcp_analysis is bespoke and routes via mcp-api")
	}
	if b.ForTool("get_custom_signal_for_skill") != mcpC {
		t.Fatal("get_custom_signal_for_skill should use mcp-api")
	}
	if b.ForTool("get_custom_strategy_definitions") != catC {
		t.Fatal("get_custom_strategy_definitions should use catalog-api directly")
	}
	if b.ForTool("create_competitor_prompt_template") != catC {
		t.Fatal("competitor prompt CRUD should use catalog-api")
	}
	if b.ForTool("get_position") != mcpC {
		t.Fatal("bot tools should use mcp-api")
	}
	if b.ForTool("get_single_prompt_template") != mcpC {
		t.Fatal("get_single_prompt_template should use mcp-api (QT token resolution on Bot)")
	}
}
