package tools

import (
	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/tools/catalog"
)

// HTTPBackends routes tool HTTP calls to GeeGoo 3xxx services.
type HTTPBackends struct {
	MCP           *mcp.Client // GeeGooBot mcp-api :3120
	SignalAPI     *mcp.Client // GeeGooSignal signal-api :3200
	SignalCatalog *mcp.Client // GeeGooSignal catalog-api :3210
	SignalAnalyze *mcp.Client // GeeGooSignal analyze-api :3230
}

// AnalysisClient prefers analyze-api (:3230) when configured.
func (b HTTPBackends) AnalysisClient() *mcp.Client {
	if b.SignalAnalyze != nil {
		return b.SignalAnalyze
	}
	return b.MCP
}

// HasMCPFallback reports whether analyze-api tools can retry mcp-api :3120.
func (b HTTPBackends) HasMCPFallback(name string) bool {
	switch name {
	case "generate_grid_strategy", "generate_dca_strategy":
		return b.SignalAnalyze != nil && b.MCP != nil && b.ForTool(name) != b.MCP
	default:
		return false
	}
}
func (b HTTPBackends) ForTool(name string) *mcp.Client {
	if catalog.UsesSignalCatalog(name) {
		if b.SignalCatalog != nil {
			return b.SignalCatalog
		}
	}
	switch name {
	case "search_code", "loopback_strategy",
		"probe_bot_signal", "probe_bot_signal_series",
		"get_indicator_series",
		"list_strategy_backtest_logs", "get_strategy_backtest_log",
		"run_strategy_backtest":
		if b.SignalAPI != nil {
			return b.SignalAPI
		}
	case "get_index_signals", "get_signal_combinations":
		if b.SignalCatalog != nil {
			return b.SignalCatalog
		}
	case "generate_grid_strategy", "generate_dca_strategy":
		if c := b.AnalysisClient(); c != nil {
			return c
		}
	}
	if b.MCP != nil {
		return b.MCP
	}
	return b.SignalAPI
}
