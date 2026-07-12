package tools

import "github.com/ghsemail/GeeGooAgent/internal/clients/mcp"

// HTTPBackends routes tool HTTP calls to GeeGoo 3xxx services.
type HTTPBackends struct {
	MCP           *mcp.Client // GeeGooBot mcp-api :3120
	SignalAPI     *mcp.Client // GeeGooSignal signal-api :3200
	SignalCatalog *mcp.Client // GeeGooSignal catalog-api :3210
	SignalAnalyze *mcp.Client // GeeGooSignal analyze-api :3230
}

// ForTool picks the Go backend for a catalog HTTP tool.
func (b HTTPBackends) ForTool(name string) *mcp.Client {
	switch name {
	case "search_code", "loopback_strategy":
		if b.SignalAPI != nil {
			return b.SignalAPI
		}
	case "get_index_signals", "get_signal_combinations":
		if b.SignalCatalog != nil {
			return b.SignalCatalog
		}
	case "get_mcp_analysis":
		if b.MCP != nil {
			return b.MCP
		}
	}
	if b.MCP != nil {
		return b.MCP
	}
	return b.SignalAPI
}
