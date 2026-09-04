// Package domaincatalog maps chat domains to playbook skills and canonical toolsets.
// Tool names are derived from internal/tools toolsets (SSOT); do not duplicate lists in cognition.
package domaincatalog

import (
	"sort"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// Domain is the chat capability bucket id (matches cognition.Domain string values).
type Domain string

const (
	DomainChat            Domain = "chat"
	DomainStockAnalysis   Domain = "stock_analysis"
	DomainNews            Domain = "news"
	DomainKnowledge       Domain = "knowledge"
	DomainReportLookup    Domain = "report_lookup"
	DomainReportWrite     Domain = "report_write"
	DomainBotManage       Domain = "bot_manage"
	DomainSignalProbe     Domain = "signal_probe"
	DomainBacktestRun     Domain = "backtest_run"
	DomainBacktestHistory Domain = "backtest_history"
	DomainCustomSignal    Domain = "custom_signal"
	DomainPromptAdmin     Domain = "prompt_admin"
	DomainDCAGrid         Domain = "dca_grid"
	DomainAmbiguous       Domain = "ambiguous"
)

// PlanSpec is the catalog view of a per-turn routing decision.
type PlanSpec struct {
	Domain          Domain
	Act             string
	Mode            string
	Confidence      float64
	Reason          string
	Skills          []string
	ToolsAllow      []string
	ClarifyQuestion string
	ClarifyChoices  []string
}

// Plugin is one chat capability bucket aligned with a playbook SKILL.md.
type Plugin struct {
	Domain   Domain
	Skills   []string
	Toolsets []string
	Act      string
	Mode     string
	Reason   string
	Conf     float64
}

var plugins = []Plugin{
	{
		Domain: DomainStockAnalysis, Skills: []string{"stock-analysis"},
		Toolsets: []string{"market", "analyst_runtime"},
		Act: "analyze", Mode: "gather", Reason: "个股分析/行情", Conf: 0.86,
	},
	{
		Domain: DomainSignalProbe, Skills: []string{"strategy-signal-probe"},
		Toolsets: []string{"strategy", "custom_signal", "market"},
		Act: "probe", Mode: "execute", Reason: "只测买卖点，不跑 PnL", Conf: 0.9,
	},
	{
		Domain: DomainBacktestRun, Skills: []string{"strategy-backtest-run"},
		Toolsets: []string{"strategy", "custom_signal", "market"},
		Act: "backtest", Mode: "execute", Reason: "明确要跑 SmartTrade 回测", Conf: 0.92,
	},
	{
		Domain: DomainDCAGrid, Skills: []string{"strategy-backtest"},
		Toolsets: []string{"strategy", "market"},
		Act: "dca_grid", Mode: "execute", Reason: "DCA/网格方案，不走 SmartTrade 回测", Conf: 0.9,
	},
	{
		Domain: DomainBacktestHistory, Skills: []string{"strategy-backtest-history"},
		Toolsets: []string{"strategy"},
		Act: "lookup", Mode: "gather", Reason: "回测历史", Conf: 0.9,
	},
	{
		Domain: DomainBotManage, Skills: []string{"bot-manager"},
		Toolsets: []string{"trading_bot", "hedge_bot", "reminder_manager", "market", "strategy", "analyst_runtime", "custom_signal"},
		Act: "bot", Mode: "gather", Reason: "交易/提醒机器人", Conf: 0.88,
	},
	{
		Domain: DomainReportLookup, Skills: []string{"report-lookup"},
		Toolsets: []string{"report_query"},
		Act: "lookup", Mode: "gather", Reason: "查询报告", Conf: 0.88,
	},
	{
		Domain: DomainReportWrite, Skills: []string{"report-write"},
		Toolsets: []string{"report_write", "analyst_runtime", "market"},
		Act: "write", Mode: "execute", Reason: "写入报告", Conf: 0.88,
	},
	{
		Domain: DomainKnowledge, Skills: []string{"knowledge-base"},
		Toolsets: []string{"knowledge"},
		Act: "lookup", Mode: "gather", Reason: "知识库检索", Conf: 0.9,
	},
	{
		Domain: DomainNews, Skills: []string{},
		Toolsets: []string{"market"},
		Act: "lookup", Mode: "gather", Reason: "新闻检索", Conf: 0.82,
	},
	{
		Domain: DomainPromptAdmin, Skills: []string{"prompt-template-admin"},
		Toolsets: []string{"prompt_admin", "analyst_runtime"},
		Act: "admin", Mode: "execute", Reason: "Prompt 模板运营", Conf: 0.86,
	},
	{
		Domain: DomainCustomSignal, Skills: []string{"custom-signal-admin"},
		Toolsets: []string{"custom_signal"},
		Act: "admin", Mode: "gather", Reason: "定制信号管理", Conf: 0.84,
	},
	{
		Domain: DomainChat, Skills: []string{},
		Toolsets: []string{"market"},
		Act: "qa", Mode: "talk", Reason: "问答/解释，不调用业务工具", Conf: 0.9,
	},
}

// PluginFor returns the catalog entry for a domain.
func PluginFor(d Domain) Plugin {
	for _, p := range plugins {
		if p.Domain == d {
			return p
		}
	}
	return Plugin{
		Domain: d, Act: "qa", Mode: "talk",
		Toolsets: []string{"market"}, Reason: "未匹配业务意图", Conf: 0.7,
	}
}

// ToolsAllow returns sorted tool names for a domain from toolset SSOT.
func ToolsAllow(d Domain) []string {
	p := PluginFor(d)
	names := tools.ChatToolNamesForToolsets(p.Toolsets)
	if d == DomainChat {
		names = filterChatTalkTools(names)
	}
	return uniq(names)
}

// filterChatTalkTools keeps only safe read-only market tools for pure chat.
func filterChatTalkTools(names []string) []string {
	allow := map[string]struct{}{
		"search_code": {}, "get_current_price": {}, "check_trading_day": {},
		"web_search": {}, "fetch_market_news": {}, "fetch_stock_news": {},
	}
	out := make([]string, 0, len(allow))
	for _, n := range names {
		if _, ok := allow[n]; ok {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

func uniq(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// MakePlan builds a PlanSpec from the catalog plugin for a domain.
func MakePlan(d Domain) PlanSpec {
	p := PluginFor(d)
	if d == DomainAmbiguous {
		return PlanSpec{
			Domain:          DomainAmbiguous,
			Act:             "disambiguate",
			Mode:            "clarify",
			Confidence:      0.55,
			Reason:          "提到策略/信号但未说明要分析、测点还是回测",
			ToolsAllow:      []string{"clarify"},
			ClarifyQuestion: "你是想做哪一件？",
			ClarifyChoices:  []string{"个股/指标分析", "测买卖点", "跑回测看收益", "先问答，先不操作"},
		}
	}
	allow := append([]string{"clarify"}, ToolsAllow(d)...)
	return PlanSpec{
		Domain:     p.Domain,
		Act:        p.Act,
		Mode:       p.Mode,
		Confidence: p.Conf,
		Reason:     p.Reason,
		Skills:     append([]string(nil), p.Skills...),
		ToolsAllow: allow,
	}
}
