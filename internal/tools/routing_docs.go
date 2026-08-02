package tools

import (
	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
)

// RoutingDoc describes a system-prompt routing guide linked to one or more tools.
type RoutingDoc struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	Source      string `json:"source"`
	InjectScope string `json:"inject_scope"`
	Content     string `json:"content"`
}

const (
	routingDocAnalysis = "analysis_routing"
)

// AllRoutingDocs returns routing guides exposed to Cockpit / dashboard UIs.
func AllRoutingDocs() []RoutingDoc {
	return []RoutingDoc{
		{
			ID:          routingDocAnalysis,
			Title:       "MCP 分析路由",
			Summary:     "技术分析 / 指标分析 / 基本面：模板选用与 get_mcp_analysis 调用决策树（含 tag=price|kline|flag|capital_flow）",
			Source:      "internal/chatprompt/analysis_routing.md",
			InjectScope: "system_prompt",
			Content:     chatprompt.AnalysisRouting(),
		},
	}
}

func routingDocIDsForTool(name string) []string {
	switch name {
	case "get_mcp_analysis", "get_single_prompt_template", "get_single_prompt_template_by_index":
		return []string{routingDocAnalysis}
	default:
		return nil
	}
}
