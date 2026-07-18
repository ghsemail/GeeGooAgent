package tools

import (
	"fmt"
	"sort"
	"strings"
)

func botCRUD(slug string) []string {
	return []string{
		"create_" + slug,
		"update_" + slug,
		"delete_" + slug,
		"list_" + slug + "s",
		"get_" + slug + "_log",
	}
}

func reportCRUD(slug string, withCreate bool) []string {
	names := []string{
		"update_" + slug + "_report",
		"delete_" + slug + "_report",
		"get_" + slug + "_reports",
	}
	if withCreate {
		names = append(names, "create_"+slug+"_report")
	}
	return names
}

var (
	botManagerTools = union(
		botCRUD("dca_bot"),
		botCRUD("grid_bot"),
		botCRUD("smart_trade"),
		botCRUD("hdg_bot"),
	)
	reminderManagerTools = union(
		botCRUD("dca_reminder"),
		botCRUD("grid_reminder"),
		botCRUD("smart_reminder"),
	)
	reportQueryTools = union(
		reportCRUD("pre_market", false),
		reportCRUD("intraday", true),
		reportCRUD("post_market", true),
		[]string{"get_stock_daily_reports", "list_today_reports"},
	)
	reportWorkflowTools = map[string]struct{}{
		"get_report_bot_codes":           {},
		"create_pre_market_report":       {},
		"save_local_report":              {},
		"write_execution_log":            {},
		"read_working_state":             {},
		"recall_yesterday_summary":       {},
		"get_bot_yesterday_attitude":     {},
		"list_today_post_market_reports": {},
	}
	promptTemplateTools = map[string]struct{}{
		"create_competitor_prompt_template": {},
		"edit_competitor_prompt_template":   {},
		"delete_competitor_prompt_template": {},
		"create_etf_prompt_template":        {},
		"edit_etf_prompt_template":          {},
		"delete_etf_prompt_template":        {},
	}
	marketTools = map[string]struct{}{
		"check_trading_day":        {},
		"search_code":              {},
		"get_current_price":        {},
		"get_ticker":               {},
		"get_position":             {},
		"get_broker":               {},
		"get_capital_flow":         {},
		"get_capital_distribution": {},
		"get_bot_yesterday_attitude": {},
		"get_mcp_analysis":         {},
		"get_single_prompt_template": {},
		"get_index_signals":        {},
		"get_signal_combinations":  {},
		"get_bot_log_by_type":      {},
		"fetch_market_news":        {},
		"fetch_stock_news":         {},
		"web_search":               {},
		"recall":                   {},
		"delegate_task":            {},
	}
	strategyTools = map[string]struct{}{
		"generate_grid_strategy": {},
		"generate_dca_strategy":  {},
		"loopback_strategy":      {},
	}
)

func union(slices ...[]string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, slice := range slices {
		for _, name := range slice {
			out[name] = struct{}{}
		}
	}
	return out
}

// ChatToolNames are on-demand tools for geegoo chat (default chat toolsets).
var ChatToolNames = ChatToolNamesForToolsets(nil)

// ToolDomain groups tools by business purpose (display alias of toolset).
type ToolDomain string

const (
	DomainReportWorkflow  ToolDomain = "report_workflow"
	DomainReportQuery     ToolDomain = "report_query"
	DomainBotManager      ToolDomain = "bot_manager"
	DomainReminderManager ToolDomain = "reminder_manager"
	DomainMarket          ToolDomain = "market"
	DomainStrategy        ToolDomain = "strategy"
	DomainPromptTemplate  ToolDomain = "prompt_template"
	DomainMeta            ToolDomain = "meta"
)

var domainLabels = map[ToolDomain]string{
	DomainReportWorkflow:  "报告 Workflow（盘前/盘中/盘后自动化，勿用于查 Bot 列表）",
	DomainReportQuery:     "报告查询（读盘前/盘中/盘后报告）",
	DomainBotManager:      "交易 Bot（DCA/GRID/SmartTrade/HDG）",
	DomainReminderManager: "提醒 Bot（DCA/GRID/Smart 提醒）",
	DomainMarket:          "行情与分析",
	DomainStrategy:        "策略生成与回测",
	DomainPromptTemplate:  "Prompt 模板",
	DomainMeta:            "其他",
}

func toolDomain(name string) ToolDomain {
	switch {
	case inSet(name, marketTools):
		return DomainMarket
	case inSet(name, strategyTools):
		return DomainStrategy
	case inSet(name, reportWorkflowTools):
		return DomainReportWorkflow
	case inSet(name, reportQueryTools):
		return DomainReportQuery
	case inSet(name, botManagerTools):
		return DomainBotManager
	case inSet(name, reminderManagerTools):
		return DomainReminderManager
	case inSet(name, promptTemplateTools):
		return DomainPromptTemplate
	default:
		return DomainMeta
	}
}

func inSet(name string, set map[string]struct{}) bool {
	_, ok := set[name]
	return ok
}

// FormatToolsListing renders grouped tool list for /tools.
func FormatToolsListing(names []string, descriptions map[string]string) string {
	grouped := map[ToolDomain][]string{}
	for _, name := range names {
		domain := toolDomain(name)
		grouped[domain] = append(grouped[domain], name)
	}
	order := []ToolDomain{
		DomainMarket, DomainStrategy, DomainBotManager, DomainReminderManager,
		DomainReportQuery, DomainReportWorkflow, DomainPromptTemplate, DomainMeta,
	}
	var lines []string
	for _, domain := range order {
		toolNames := grouped[domain]
		if len(toolNames) == 0 {
			continue
		}
		sort.Strings(toolNames)
		lines = append(lines, fmt.Sprintf("[%s]", domainLabels[domain]))
		for _, name := range toolNames {
			desc := descriptions[name]
			if len(desc) > 72 {
				desc = desc[:72] + "…"
			}
			lines = append(lines, fmt.Sprintf("  - %s: %s", name, desc))
		}
		lines = append(lines, "")
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

// RegisteredChatToolNames returns chat allowlist tools present in registry
// using the default chat toolsets.
func RegisteredChatToolNames(registry *Registry) []string {
	return RegisteredChatToolNamesFor(registry, nil)
}
