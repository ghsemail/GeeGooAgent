package chatprompt

import _ "embed"

// analysis_routing.md is the single source of truth for tech / index / fundamental MCP routing.
//
//go:embed analysis_routing.md
var analysisRoutingMD string

// AnalysisRouting returns MCP analysis type routing rules embedded from analysis-routing.md.
func AnalysisRouting() string {
	return analysisRoutingMD
}
