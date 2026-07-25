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
	botManagerTools = mergeSets(tradingBotTools, hedgeBotTools)
	reminderManagerTools = union(
		botCRUD("dca_reminder"),
		botCRUD("grid_reminder"),
		botCRUD("smart_reminder"),
	)
	reportQueryTools = union(
		reportCRUD("pre_market", false),
		reportCRUD("intraday", true),
		reportCRUD("post_market", true),
		[]string{
			"get_stock_daily_reports", "list_today_reports",
			"get_bot_yesterday_attitude", "get_bot_log_by_type",
		},
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
	marketDataTools = map[string]struct{}{
		"check_trading_day":  {},
		"get_current_price":  {},
		"get_ticker":         {},
		"get_broker":         {},
		"get_position":       {},
	}
	researchTools = map[string]struct{}{
		"get_capital_flow":         {},
		"get_capital_distribution": {},
		"get_mcp_analysis":         {},
		"get_single_prompt_template": {},
	}
	infoSearchTools = map[string]struct{}{
		"search_code":       {},
		"fetch_market_news": {},
		"fetch_stock_news":  {},
		"web_search":        {},
	}
	agentMetaTools = map[string]struct{}{
		"recall":         {},
		"save_note":      {},
		"manage_memory":  {},
		"update_soul":    {},
		"create_skill":   {},
		"clarify":        {},
		"delegate_task":  {},
		"delegate_tasks": {},
	}
	// legacyMarketTools is the deprecated market toolset union (alias compatibility).
	legacyMarketTools = mergeSets(marketDataTools, researchTools, infoSearchTools)
	strategyTools = map[string]struct{}{
		"get_index_signals":       {},
		"get_signal_combinations": {},
		"generate_grid_strategy":    {},
		"generate_dca_strategy":   {},
		"loopback_strategy":       {},
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
	DomainReportWorkflow  ToolDomain = "report_workflow"
	DomainReportQuery     ToolDomain = "report_query"
	DomainTradingBot      ToolDomain = "trading_bot"
	DomainHedgeBot        ToolDomain = "hedge_bot"
	DomainReminderManager ToolDomain = "reminder_manager"
	DomainMarketData      ToolDomain = "market_data"
	DomainResearch        ToolDomain = "research"
	DomainInfoSearch      ToolDomain = "info_search"
	DomainAgentMeta       ToolDomain = "agent_meta"
	DomainStrategy        ToolDomain = "strategy"
	DomainPromptTemplate  ToolDomain = "prompt_template"
	DomainMeta            ToolDomain = "meta"
)

var domainShortLabels = map[ToolDomain]string{
	DomainMarketData:      "行情与账户",
	DomainResearch:        "研究与分析",
	DomainInfoSearch:      "信息检索",
	DomainAgentMeta:       "Agent 元能力",
	DomainStrategy:        "策略与信号",
	DomainTradingBot:      "交易机器人",
	DomainHedgeBot:        "对冲机器人",
	DomainReminderManager: "提醒机器人",
	DomainReportQuery:     "报告查询",
	DomainReportWorkflow:  "报告 Workflow",
	DomainPromptTemplate:  "Prompt 模板",
	DomainMeta:            "其他",
}

var domainDescriptions = map[ToolDomain]string{
	DomainReportWorkflow:  "盘前/盘中/盘后自动化写报告（勿用于查 Bot 列表）",
	DomainReportQuery:     "读报告、Bot 态度与运行日志",
	DomainTradingBot:      "DCA / GRID / SmartTrade 读写",
	DomainHedgeBot:        "HDG 对冲机器人读写",
	DomainReminderManager: "DCA / GRID / Smart 提醒读写",
	DomainMarketData:      "交易日、现价、逐笔、席位、持仓",
	DomainResearch:        "资金面、MCP 技术分析、Prompt 模板",
	DomainInfoSearch:      "搜码、市场/个股新闻、网页搜索",
	DomainAgentMeta:       "记忆、澄清、委派等横切能力",
	DomainStrategy:        "信号列表、网格/DCA 生成与回测",
	DomainPromptTemplate:  "竞品/ETF 分析模板 CRUD",
	DomainMeta:            "未归类的注册工具",
}

// domainLabels kept for CLI /tools listing compatibility.
var domainLabels = map[ToolDomain]string{
	DomainReportWorkflow:  "报告 Workflow（盘前/盘中/盘后自动化，勿用于查 Bot 列表）",
	DomainReportQuery:     "报告查询（读报告、Bot 态度与日志）",
	DomainTradingBot:      "交易机器人（DCA/GRID/SmartTrade）",
	DomainHedgeBot:        "对冲机器人（HDG）",
	DomainReminderManager: "提醒机器人（DCA/GRID/Smart 提醒）",
	DomainMarketData:      "行情与账户",
	DomainResearch:        "研究与分析",
	DomainInfoSearch:      "信息检索",
	DomainAgentMeta:       "Agent 元能力",
	DomainStrategy:        "策略与信号",
	DomainPromptTemplate:  "Prompt 模板",
	DomainMeta:            "其他",
}

var domainOrderKeys = []ToolDomain{
	DomainMarketData, DomainResearch, DomainInfoSearch, DomainStrategy,
	DomainTradingBot, DomainHedgeBot, DomainReminderManager,
	DomainReportQuery, DomainReportWorkflow, DomainAgentMeta, DomainPromptTemplate, DomainMeta,
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
		name == "clarify", name == "delegate_task", name == "update_soul", name == "create_skill":
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
	case inSet(name, marketDataTools):
		return DomainMarketData
	case inSet(name, researchTools):
		return DomainResearch
	case inSet(name, infoSearchTools):
		return DomainInfoSearch
	case inSet(name, agentMetaTools):
		return DomainAgentMeta
	case inSet(name, strategyTools):
		return DomainStrategy
	case inSet(name, reportWorkflowTools):
		return DomainReportWorkflow
	case inSet(name, reportQueryTools):
		return DomainReportQuery
	case inSet(name, tradingBotTools):
		return DomainTradingBot
	case inSet(name, hedgeBotTools):
		return DomainHedgeBot
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
		DomainMarketData, DomainResearch, DomainInfoSearch, DomainStrategy,
		DomainTradingBot, DomainHedgeBot, DomainReminderManager,
		DomainReportQuery, DomainReportWorkflow, DomainAgentMeta, DomainPromptTemplate, DomainMeta,
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
