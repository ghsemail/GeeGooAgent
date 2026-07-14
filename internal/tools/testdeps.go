package tools

import "github.com/ghsemail/GeeGooAgent/internal/clients/mcp"

// TestHTTPBackends wires one client to every backend (unit tests).
func TestHTTPBackends(client *mcp.Client) HTTPBackends {
	if client == nil {
		return HTTPBackends{}
	}
	return HTTPBackends{MCP: client, SignalAPI: client, SignalCatalog: client, SignalAnalyze: client}
}
