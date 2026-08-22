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
	tradingBotTools = union(
		botCRUD("dca_bot"),
		botCRUD("grid_bot"),
		botCRUD("smart_trade"),
	)
	hedgeBotTools = union(
		botCRUD("hdg_bot"),
	)
	// botManagerTools is the legacy union (bot_manager alias).
	botManagerTools      = mergeSets(tradingBotTools, hedgeBotTools)
	reminderManagerTools = union(
		botCRUD("dca_reminder"),
		botCRUD("grid_reminder"),
		botCRUD("smart_reminder"),
	)
	reportQueryTools = union(
		[]string{
			"get_stock_premarket_reports",
			"get_stock_intraday_reports",
			"get_stock_postmarket_reports",
			"get_stock_daily_reports",
			"list_today_reports",
			"get_bot_yesterday_attitude",
			"get_bot_log_by_type",
		},
	)
	reportWriteTools = union(
		[]string{
			"create_stock_intraday_report",
			"update_stock_intraday_report",
			"delete_stock_intraday_report",
			"create_stock_postmarket_report",
			"update_stock_postmarket_report",
			"delete_stock_postmarket_report",
			"update_stock_premarket_report",
			"delete_stock_premarket_report",
		},
	)
	reportWorkflowTools = map[string]struct{}{
		"get_report_bot_codes":                {},
		"create_market_premarket_report":      {},
		"get_market_premarket_report":         {},
		"create_stock_premarket_report":       {},
		"save_local_report":                   {},
		"write_execution_log":                 {},
		"read_working_state":                  {},
		"recall_yesterday_summary":            {},
		"list_today_stock_postmarket_reports": {},
	}
	analystRuntimeTools = map[string]struct{}{
		"get_single_prompt_template":          {},
		"get_single_prompt_template_by_index": {},
		"get_mcp_analysis":                    {},
		"get_hourly_analysis_bundle":          {},
		"get_capital_flow":                    {},
		"get_capital_distribution":            {},
	}
	promptAdminTools = map[string]struct{}{
		"add_single_prompt_template":        {},
		"edit_prompt_template":              {},
		"delete_prompt_template":            {},
		"switch_prompt_status":              {},
		"create_competitor_prompt_template": {},
		"edit_competitor_prompt_template":   {},
		"delete_competitor_prompt_template": {},
		"create_etf_prompt_template":        {},
		"edit_etf_prompt_template":          {},
		"delete_etf_prompt_template":        {},
	}
	customSignalTools = map[string]struct{}{
		"get_custom_signal":               {},
		"get_custom_signal_for_skill":     {},
		"get_all_custom_signal_id":        {},
		"get_custom_strategy_definitions": {},
		"add_custom_signal":               {},
		"edit_custom_signal":              {},
		"delete_custom_signal":            {},
	}
	marketTools = map[string]struct{}{
		"check_trading_day": {},
		"get_current_price": {},
		"get_ticker":        {},
		"get_broker":        {},
		"get_position":      {},
		"search_code":       {},
		"fetch_market_news": {},
		"fetch_stock_news":  {},
		"web_search":        {},
	}
	agentMetaTools = map[string]struct{}{
		"recall":            {},
		"save_note":         {},
		"manage_memory":     {},
		"update_soul":       {},
		"update_preference": {},
		"create_skill":      {},
		"clarify":           {},
		"delegate_task":     {},
		"delegate_tasks":    {},
	}
	strategyTools = map[string]struct{}{
		"get_index_signals":       {},
		"get_signal_combinations": {},
		"generate_grid_strategy":  {},
		"generate_dca_strategy":   {},
		"loopback_strategy":       {},
	}
	knowledgeTools = map[string]struct{}{
		"search_knowledge": {},
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

func mergeSets(sets ...map[string]struct{}) map[string]struct{} {
	out := map[string]struct{}{}
	for _, set := range sets {
		for name := range set {
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
	DomainMarket          ToolDomain = "market"
	DomainAnalystRuntime  ToolDomain = "analyst_runtime"
	DomainPromptAdmin     ToolDomain = "prompt_admin"
	DomainCustomSignal    ToolDomain = "custom_signal"
	DomainStrategy        ToolDomain = "strategy"
	DomainReportQuery     ToolDomain = "report_query"
	DomainReportWrite     ToolDomain = "report_write"
	DomainReportWorkflow  ToolDomain = "report_workflow"
	DomainTradingBot      ToolDomain = "trading_bot"
	DomainHedgeBot        ToolDomain = "hedge_bot"
	DomainReminderManager ToolDomain = "reminder_manager"
	DomainAgentMeta       ToolDomain = "agent_meta"
	DomainKnowledge       ToolDomain = "knowledge"
	DomainMeta            ToolDomain = "meta"
)

var domainShortLabels = map[ToolDomain]string{
	DomainMarket:          "行情与资金",
	DomainAnalystRuntime:  "运行时分析",
	DomainPromptAdmin:     "模板运营",
	DomainCustomSignal:    "定制策略",
	DomainStrategy:        "策略与回测",
	DomainReportQuery:     "报告查询",
	DomainReportWrite:     "报告写入",
	DomainReportWorkflow:  "报告 Workflow",
	DomainTradingBot:      "交易机器人",
	DomainHedgeBot:        "对冲机器人",
	DomainReminderManager: "提醒机器人",
	DomainAgentMeta:       "Agent 元能力",
	DomainKnowledge:       "知识库",
	DomainMeta:            "其他",
}

var domainDescriptions = map[ToolDomain]string{
	DomainMarket:          "交易日、搜码、行情快照、逐笔、席位、持仓、新闻检索",
	DomainAnalystRuntime:  "运行时分析：读 single_prompt_template 启用列表（供 get_mcp_analysis 选 prompt_id）",
	DomainPromptAdmin:     "模板运营：写 single_prompt_template / 竞品·ETF 用户模板（改完后需 switch 启用才出现在运行时列表）",
	DomainCustomSignal:    "定制策略定义与 CRUD（Monday · signal_custom_db · :3210）",
	DomainStrategy:        "信号列表、网格/DCA 生成与回测",
	DomainReportQuery:     "读已有报告、Bot 态度与运行日志",
	DomainReportWrite:     "Chat 中补写/修改盘前盘中盘后报告",
	DomainReportWorkflow:  "盘前/盘后自动化流水线（勿用于查 Bot 列表）",
	DomainTradingBot:      "DCA / GRID / SmartTrade 读写",
	DomainHedgeBot:        "HDG 对冲机器人读写",
	DomainReminderManager: "DCA / GRID / Smart 提醒读写",
	DomainAgentMeta:       "记忆、澄清、委派等横切能力",
	DomainKnowledge:       "WeKnora 知识库检索（按 skill 注入，不进默认 chat）",
	DomainMeta:            "未归类的注册工具",
}

// domainLabels kept for CLI /tools listing compatibility.
var domainLabels = map[ToolDomain]string{
	DomainMarket:          "行情与资金",
	DomainAnalystRuntime:  "运行时分析",
	DomainPromptAdmin:     "模板运营（高级）",
	DomainCustomSignal:    "定制策略",
	DomainStrategy:        "策略与回测",
	DomainReportQuery:     "报告查询",
	DomainReportWrite:     "报告写入",
	DomainReportWorkflow:  "报告 Workflow（自动化专用）",
	DomainTradingBot:      "交易机器人（DCA/GRID/SmartTrade）",
	DomainHedgeBot:        "对冲机器人（HDG）",
	DomainReminderManager: "提醒机器人（DCA/GRID/Smart）",
	DomainAgentMeta:       "Agent 元能力",
	DomainKnowledge:       "知识库",
	DomainMeta:            "其他",
}

var domainOrderKeys = []ToolDomain{
	DomainMarket, DomainAnalystRuntime, DomainPromptAdmin, DomainCustomSignal, DomainStrategy,
	DomainTradingBot, DomainHedgeBot, DomainReminderManager,
	DomainReportQuery, DomainReportWrite, DomainReportWorkflow, DomainAgentMeta, DomainKnowledge, DomainMeta,
}

// DomainOrder returns stable display order for a tool domain (1-based).
func DomainOrder(d ToolDomain) int {
	for i, key := range domainOrderKeys {
		if key == d {
			return i + 1
		}
	}
	return len(domainOrderKeys) + 1
}

// DomainShortLabel returns a concise UI label for the domain.
func DomainShortLabel(d ToolDomain) string {
	if s, ok := domainShortLabels[d]; ok {
		return s
	}
	return string(d)
}

// DomainDescription returns a longer hint for tooltips.
func DomainDescription(d ToolDomain) string {
	if s, ok := domainDescriptions[d]; ok {
		return s
	}
	return domainLabels[d]
}

// ToolTaxonomy is the cognitive-flow grouping (perceive → act).
type ToolTaxonomy string

const (
	TaxonomyPerceive ToolTaxonomy = "perceive"
	TaxonomyAnalyze  ToolTaxonomy = "analyze"
	TaxonomyDecide   ToolTaxonomy = "decide"
	TaxonomyAct      ToolTaxonomy = "act"
	TaxonomyMeta     ToolTaxonomy = "meta"
	TaxonomyOther    ToolTaxonomy = "other"
)

var taxonomyLabels = map[ToolTaxonomy]string{
	TaxonomyPerceive: "感知 Perception",
	TaxonomyAnalyze:  "分析 Analysis",
	TaxonomyDecide:   "决策 Decision",
	TaxonomyAct:      "执行 Action",
	TaxonomyMeta:     "元能力 Meta",
	TaxonomyOther:    "其他",
}

var taxonomyOrderKeys = []ToolTaxonomy{
	TaxonomyPerceive, TaxonomyAnalyze, TaxonomyDecide, TaxonomyAct, TaxonomyMeta, TaxonomyOther,
}

func toolTaxonomy(name string) ToolTaxonomy {
	switch {
	case strings.HasPrefix(name, "search_"), name == "recall", strings.HasPrefix(name, "fetch_"),
		name == "web_search", name == "check_trading_day":
		return TaxonomyPerceive
	case strings.HasPrefix(name, "get_"), strings.HasPrefix(name, "list_"), strings.Contains(name, "analysis"):
		return TaxonomyAnalyze
	case strings.HasPrefix(name, "generate_"), strings.HasPrefix(name, "loopback"):
		return TaxonomyDecide
	case strings.HasPrefix(name, "create_"), strings.HasPrefix(name, "update_"), strings.HasPrefix(name, "delete_"):
		return TaxonomyAct
	case strings.HasPrefix(name, "read_"), name == "write_execution_log",
		strings.HasPrefix(name, "save_"), strings.HasPrefix(name, "manage_"),
		name == "clarify", name == "delegate_task", name == "update_soul", name == "update_preference", name == "create_skill":
		return TaxonomyMeta
	default:
		return TaxonomyOther
	}
}

// TaxonomyOrder returns stable display order for taxonomy (1-based).
func TaxonomyOrder(t ToolTaxonomy) int {
	for i, key := range taxonomyOrderKeys {
		if key == t {
			return i + 1
		}
	}
	return len(taxonomyOrderKeys) + 1
}

// TaxonomyLabel returns the display label for a taxonomy id.
func TaxonomyLabel(t ToolTaxonomy) string {
	if s, ok := taxonomyLabels[t]; ok {
		return s
	}
	return string(t)
}

func toolDomain(name string) ToolDomain {
	switch {
	case inSet(name, marketTools):
		return DomainMarket
	case inSet(name, analystRuntimeTools):
		return DomainAnalystRuntime
	case inSet(name, promptAdminTools):
		return DomainPromptAdmin
	case inSet(name, customSignalTools):
		return DomainCustomSignal
	case inSet(name, strategyTools):
		return DomainStrategy
	case inSet(name, reportWorkflowTools):
		return DomainReportWorkflow
	case inSet(name, reportQueryTools):
		return DomainReportQuery
	case inSet(name, reportWriteTools):
		return DomainReportWrite
	case inSet(name, agentMetaTools):
		return DomainAgentMeta
	case inSet(name, tradingBotTools):
		return DomainTradingBot
	case inSet(name, hedgeBotTools):
		return DomainHedgeBot
	case inSet(name, reminderManagerTools):
		return DomainReminderManager
	case inSet(name, knowledgeTools):
		return DomainKnowledge
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
	order := domainOrderKeys
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
