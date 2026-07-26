package tools

import (
	"sort"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/tools/catalog"
)

// CatalogItem is a rich tool descriptor for Cockpit / dashboard UIs.
type CatalogItem struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Domain            string   `json:"domain"`
	DomainLabel       string   `json:"domain_label"`
	DomainShortLabel  string   `json:"domain_short_label"`
	DomainOrder       int      `json:"domain_order"`
	DomainDescription string   `json:"domain_description"`
	Taxonomy          string   `json:"taxonomy"`
	TaxonomyLabel     string   `json:"taxonomy_label"`
	TaxonomyOrder     int      `json:"taxonomy_order"`
	Toolsets          []string `json:"toolsets"`
	ChatEnabled       bool     `json:"chat_enabled"`
	RequiresMCP       bool     `json:"requires_mcp"`
	RequiresApproval  bool     `json:"requires_approval"`
	Implementation    string   `json:"implementation"` // http | bespoke
	HTTPPath          string   `json:"http_path,omitempty"`
	ContextSummary    string   `json:"context_summary"`
	ContextInjections []string `json:"context_injections"`
}

// ToolsetSummary describes a builtin toolset for UI grouping.
type ToolsetSummary struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	ChatDefault bool   `json:"chat_default"`
	ToolCount   int    `json:"tool_count"`
}

// TaxonomySummary describes a cognitive-flow tool group for UI.
type TaxonomySummary struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Order int    `json:"order"`
}

// BuildCatalog builds tool metadata from the live registry and chat allowlist.
func BuildCatalog(registry *Registry, chatNames []string) []CatalogItem {
	if registry == nil {
		return nil
	}
	chatSet := map[string]struct{}{}
	for _, name := range chatNames {
		chatSet[name] = struct{}{}
	}
	httpPaths := map[string]string{}
	httpMCP := map[string]bool{}
	for _, spec := range catalog.AllHTTP() {
		httpPaths[spec.Name] = spec.Path
		if spec.RequiresMCPToken {
			httpMCP[spec.Name] = true
		} else {
			httpMCP[spec.Name] = catalog.NeedsMCPToken(spec.Name)
		}
	}

	items := make([]CatalogItem, 0, len(registry.ListNames()))
	for _, name := range registry.ListNames() {
		tool, ok := registry.Get(name)
		if !ok {
			continue
		}
		domain := toolDomain(name)
		tax := toolTaxonomy(name)
		_, chat := chatSet[name]
		impl := "bespoke"
		path := ""
		reqMCP := requiresMCPToken(name)
		if p, ok := httpPaths[name]; ok {
			impl = "http"
			path = p
			reqMCP = httpMCP[name]
		}
		injections, summary := contextForTool(name, impl, reqMCP)
		items = append(items, CatalogItem{
			Name:              tool.Name,
			Description:       tool.Description,
			Domain:            string(domain),
			DomainLabel:       domainLabels[domain],
			DomainShortLabel:  DomainShortLabel(domain),
			DomainOrder:       DomainOrder(domain),
			DomainDescription: DomainDescription(domain),
			Taxonomy:          string(tax),
			TaxonomyLabel:     TaxonomyLabel(tax),
			TaxonomyOrder:     TaxonomyOrder(tax),
			Toolsets:          toolsetIDsFor(name),
			ChatEnabled:       chat,
			RequiresMCP:       reqMCP,
			RequiresApproval:  ApprovalRequired(name),
			Implementation:    impl,
			HTTPPath:          path,
			ContextSummary:    summary,
			ContextInjections: injections,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].DomainOrder != items[j].DomainOrder {
			return items[i].DomainOrder < items[j].DomainOrder
		}
		return items[i].Name < items[j].Name
	})
	return items
}

// BuildToolsetSummaries returns toolset metadata for UI sidebars.
func BuildToolsetSummaries() []ToolsetSummary {
	out := make([]ToolsetSummary, 0, len(builtinToolsets))
	for _, ts := range builtinToolsets {
		out = append(out, ToolsetSummary{
			ID:          ts.ID,
			Label:       ts.Label,
			Description: ts.Description,
			ChatDefault: ts.ChatDefault,
			ToolCount:   len(ts.names),
		})
	}
	return out
}

// BuildTaxonomySummaries returns taxonomy metadata for UI grouping.
func BuildTaxonomySummaries() []TaxonomySummary {
	out := make([]TaxonomySummary, 0, len(taxonomyOrderKeys))
	for _, tax := range taxonomyOrderKeys {
		out = append(out, TaxonomySummary{
			ID:    string(tax),
			Label: TaxonomyLabel(tax),
			Order: TaxonomyOrder(tax),
		})
	}
	return out
}

func toolsetIDsFor(name string) []string {
	var ids []string
	for _, ts := range builtinToolsets {
		if ts.Contains(name) {
			ids = append(ids, ts.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func requiresMCPToken(name string) bool {
	switch name {
	case "search_code", "web_search", "get_index_signals", "get_signal_combinations", "clarify":
		return false
	}
	if strings.HasPrefix(name, "create_") && strings.Contains(name, "prompt_template") {
		return true
	}
	switch name {
	case "save_note", "manage_memory", "recall", "update_soul", "create_skill":
		return false
	}
	return catalog.NeedsMCPToken(name) || catalog.BespokeNames[name]
}

func contextForTool(name, impl string, requiresMCP bool) (injections []string, summary string) {
	injections = []string{"session_history"}
	summaryParts := []string{"会话历史与当前轮 step 记录会作为 Working Memory 提供给模型。"}

	switch name {
	case "save_note", "manage_memory", "update_soul", "create_skill":
		injections = append(injections, "tenant_user_id", "postgres_facts")
		summaryParts = append(summaryParts, "按登录租户 user_id 读写长期记忆（facts / SOUL / skills）。")
	case "recall":
		injections = append(injections, "tenant_user_id", "postgres_facts", "postgres_episodes")
		summaryParts = append(summaryParts, "按 user_id 检索 facts + episodes；默认 Chat 已用 Retrieval Gate 自动注入，此工具主要给 Workflow。")
	case "clarify":
		injections = append(injections, "user_input_channel")
		summaryParts = append(summaryParts, "通过 Chat UI 阻塞等待用户选择/输入，结果回填到当前轮。")
	case "delegate_task":
		injections = append(injections, "sub_agent_context")
		summaryParts = append(summaryParts, "派生子 Agent，继承 session_id 与工具上下文。")
	case "read_working_state", "write_execution_log", "save_local_report", "recall_yesterday_summary":
		injections = append(injections, "workspace_files", "session_id")
		summaryParts = append(summaryParts, "读写 workspace 工作区文件，用于 report_workflow 自动化。")
	case "get_mcp_analysis":
		if requiresMCP {
			injections = append(injections, "mcp_token", "trading_user_id")
			summaryParts = append(summaryParts, "注入 agent-runtime 配置的 mcp_token，经 GeeGooBot 解析为交易 user_id 后调用 analyze-api。")
		}
	case "get_single_prompt_template":
		if requiresMCP {
			injections = append(injections, "mcp_token", "trading_user_id")
			summaryParts = append(summaryParts, "注入 mcp_token，经 GeeGooSignal catalog-api (:3210) 取可用 Prompt 列表（运行时分析前置）。")
		}
	case "get_single_prompt_template_by_index",
		"get_custom_signal", "get_custom_signal_for_skill", "get_all_custom_signal_id", "get_custom_strategy_definitions":
		if requiresMCP {
			injections = append(injections, "mcp_token", "trading_user_id")
			summaryParts = append(summaryParts, "注入 mcp_token，经 GeeGooSignal catalog-api (:3210) 鉴权后访问 Monday 模板与定制策略。")
		}
	case "list_smart_trades", "list_grid_bots", "list_dca_bots", "list_hdg_bots",
		"list_dca_reminders", "list_grid_reminders", "list_smart_reminders",
		"create_dca_bot", "create_grid_bot", "create_smart_trade", "create_hdg_bot",
		"update_dca_bot", "update_grid_bot", "update_smart_trade", "update_hdg_bot",
		"delete_dca_bot", "delete_grid_bot", "delete_smart_trade", "delete_hdg_bot",
		"get_dca_bot_log", "get_grid_bot_log", "get_smart_trade_log", "get_hdg_bot_log",
		"get_position", "get_ticker", "get_broker", "get_capital_flow", "get_capital_distribution",
		"get_bot_yesterday_attitude", "check_trading_day", "fetch_market_news", "fetch_stock_news":
		if requiresMCP {
			injections = append(injections, "mcp_token", "trading_user_id")
			summaryParts = append(summaryParts, "注入 agent-runtime 配置的 mcp_token（交易用户身份），非运营台 admin token。")
		}
	default:
		if impl == "http" && requiresMCP {
			injections = append(injections, "mcp_token", "trading_user_id")
			summaryParts = append(summaryParts, "HTTP 转发至 GeeGooBot mcp-api，body 携带 mcp_token 解析 user_id。")
		} else if impl == "bespoke" && requiresMCP {
			injections = append(injections, "mcp_token", "trading_user_id")
			summaryParts = append(summaryParts, "内置 handler 调用 MCP/Signal，携带 mcp_token。")
		}
	}

	if ApprovalRequired(name) {
		injections = append(injections, "user_approval")
		summaryParts = append(summaryParts, "Chat 写操作需用户在 Plan 卡片中确认后才会执行。")
	}

	if name != "recall" {
		summaryParts = append(summaryParts, "每轮开始前 Retrieval Gate 可能自动注入相关 facts/episodes（与工具调用无关的全局上下文）。")
	}

	seen := map[string]struct{}{}
	uniq := make([]string, 0, len(injections))
	for _, tag := range injections {
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		uniq = append(uniq, tag)
	}
	return uniq, strings.Join(summaryParts, " ")
}

// CatalogItemToMap converts CatalogItem to dashboard JSON map.
func CatalogItemToMap(item CatalogItem) map[string]any {
	return map[string]any{
		"name":                item.Name,
		"description":         item.Description,
		"domain":              item.Domain,
		"domain_label":        item.DomainLabel,
		"domain_short_label":  item.DomainShortLabel,
		"domain_order":        item.DomainOrder,
		"domain_description":  item.DomainDescription,
		"taxonomy":            item.Taxonomy,
		"taxonomy_label":      item.TaxonomyLabel,
		"taxonomy_order":      item.TaxonomyOrder,
		"toolsets":            item.Toolsets,
		"chat":                item.ChatEnabled,
		"chat_enabled":        item.ChatEnabled,
		"requires_mcp":        item.RequiresMCP,
		"requires_approval":   item.RequiresApproval,
		"implementation":      item.Implementation,
		"http_path":           item.HTTPPath,
		"context_summary":     item.ContextSummary,
		"context_injections":  item.ContextInjections,
		"source":              item.Implementation,
	}
}
