package cognition

import "strings"

// RulePlanner classifies turns with domain rules. Gray-zone utterances
// become clarify/talk instead of silently executing a playbook.
type RulePlanner struct{}

// Plan implements Planner.
func (RulePlanner) Plan(in PlanInput) TurnPlan {
	msg := strings.TrimSpace(in.UserText)
	switch {
	case isDefinitionOrChitchat(msg):
		return makePlan(DomainChat, "qa", ModeTalk, 0.9, "问答/解释，不调用业务工具", nil, toolsChat)
	case hasAny(msg, promptAdminTokens):
		return makePlan(DomainPromptAdmin, "admin", ModeExecute, 0.86, "Prompt 模板运营", []string{"prompt-template-admin"}, toolsPromptAdmin)
	case hasAny(msg, customSignalTokens):
		return makePlan(DomainCustomSignal, "admin", ModeGather, 0.84, "定制信号管理", []string{"custom-signal-admin"}, toolsCustomSignal)
	case hasAny(msg, reportWriteTokens):
		return makePlan(DomainReportWrite, "write", ModeExecute, 0.88, "写入报告", []string{"report-write"}, toolsReportWrite)
	case hasAny(msg, reportLookupTokens):
		return makePlan(DomainReportLookup, "lookup", ModeGather, 0.88, "查询报告", []string{"report-lookup"}, toolsReportLookup)
	case hasAny(msg, knowledgeTokens):
		return makePlan(DomainKnowledge, "lookup", ModeGather, 0.9, "知识库检索", []string{"knowledge-base"}, toolsKnowledge)
	case hasAny(msg, newsTokens):
		return makePlan(DomainNews, "lookup", ModeGather, 0.82, "新闻检索", nil, toolsNews)
	case hasAny(msg, botManageTokens):
		mode := ModeGather
		if hasAny(msg, []string{"建", "创建", "改", "删除", "停"}) {
			mode = ModeExecute
		}
		return makePlan(DomainBotManage, "bot", mode, 0.88, "交易/提醒机器人", []string{"bot-manager"}, toolsBotManage)
	case hasAny(msg, dcaGridTokens):
		return makePlan(DomainDCAGrid, "dca_grid", ModeExecute, 0.9, "DCA/网格方案，不走 SmartTrade 回测", []string{"strategy-backtest"}, toolsDCAGrid)
	case hasAny(msg, backtestHistoryTokens):
		return makePlan(DomainBacktestHistory, "lookup", ModeGather, 0.9, "回测历史", []string{"strategy-backtest-history"}, toolsBacktestHistory)
	case isSignalProbe(msg):
		return makePlan(DomainSignalProbe, "probe", ModeExecute, 0.9, "只测买卖点，不跑 PnL", []string{"strategy-signal-probe"}, toolsSignalProbe)
	case isBacktestRun(msg):
		return makePlan(DomainBacktestRun, "backtest", ModeExecute, 0.92, "明确要跑 SmartTrade 回测", []string{"strategy-backtest-run"}, toolsBacktestRun)
	case isBareStrategyTalk(msg):
		return TurnPlan{
			Domain:          DomainAmbiguous,
			Act:             "disambiguate",
			Mode:            ModeClarify,
			Confidence:      0.55,
			Reason:          "提到策略/信号但未说明要分析、测点还是回测",
			ToolsAllow:      []string{alwaysAllowClarify},
			ClarifyQuestion: "你是想做哪一件？",
			ClarifyChoices:  []string{"个股/指标分析", "测买卖点", "跑回测看收益", "先问答，先不操作"},
		}
	case isStockAnalysis(msg):
		return makePlan(DomainStockAnalysis, "analyze", ModeGather, 0.86, "个股分析/行情", []string{"stock-analysis"}, toolsStockAnalysis)
	default:
		return makePlan(DomainChat, "qa", ModeTalk, 0.7, "未匹配业务意图，按对话处理", nil, toolsChat)
	}
}

func makePlan(domain Domain, act string, mode Mode, conf float64, reason string, skills, tools []string) TurnPlan {
	allow := append([]string{alwaysAllowClarify}, tools...)
	return TurnPlan{
		Domain:     domain,
		Act:        act,
		Mode:       mode,
		Confidence: conf,
		Reason:     reason,
		Skills:     skills,
		ToolsAllow: uniq(allow),
	}
}

func isDefinitionOrChitchat(msg string) bool {
	return hasAny(msg, []string{
		"是什么", "什么意思", "怎么解释", "解释一下", "为什么这么说",
		"你好", "谢谢", "准吗", "靠谱吗", "怎么理解",
	})
}

func isSignalProbe(msg string) bool {
	if hasAny(msg, []string{"测信号", "买卖点", "有没有买卖", "信号密度", "只看买卖"}) {
		return true
	}
	return strings.Contains(msg, "回测") && hasAny(msg, []string{"买卖", "有没有信号"}) &&
		!hasAny(msg, []string{"收益", "回撤", "成交笔"})
}

func isBacktestRun(msg string) bool {
	if hasAny(msg, dcaGridTokens) {
		return false
	}
	if hasAny(msg, backtestHistoryTokens) {
		return false
	}
	if isSignalProbe(msg) && !hasAny(msg, []string{"收益", "回撤", "成交笔", "pnl"}) {
		return false
	}
	return hasAny(msg, []string{"回测", "跑回测", "再回测", "backtest", "就用刚才那套", "再跑回测"})
}

func isBareStrategyTalk(msg string) bool {
	if !hasAny(msg, []string{"信号", "策略"}) && !hasAny(msg, []string{"MACD", "macd", "RSI", "rsi", "SAR", "sar"}) {
		return false
	}
	if hasAny(msg, []string{"分析", "技术面", "走势", "回测", "买卖", "机器人", "报告", "知识库", "模板", "新闻"}) {
		return false
	}
	return true
}

func isStockAnalysis(msg string) bool {
	return hasAny(msg, []string{
		"分析", "技术面", "基本面", "走势", "趋势", "现价", "股价", "价格", "多少钱",
		"查一下", "看看行情", "资金面", "怎么样", "涨跌", "行情",
	})
}

func hasAny(msg string, tokens []string) bool {
	lower := strings.ToLower(msg)
	for _, tok := range tokens {
		if tok == "" {
			continue
		}
		if strings.Contains(msg, tok) || strings.Contains(lower, strings.ToLower(tok)) {
			return true
		}
	}
	return false
}

func uniq(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

var (
	promptAdminTokens  = []string{"加模板", "改 Prompt", "改Prompt", "启用模板", "禁用模板", "EMA 模板", "prompt模板", "Prompt 模板"}
	customSignalTokens = []string{"定制信号", "定制策略", "改策略参数", "custom_signal"}
	reportWriteTokens  = []string{"写成报告", "保存盘", "落库报告", "补写报告", "保存报告", "生成报告"}
	reportLookupTokens = []string{"查报告", "今日报告", "今天盘前", "盘前写了", "昨日态度", "盘后报告", "盘中报告"}
	knowledgeTokens    = []string{"知识库", "查库", "按知识库", "文档里怎么写"}
	newsTokens         = []string{"新闻", "资讯"}
	botManageTokens    = []string{
		"机器人", "我的网格", "建个网格", "建个 dca", "建个DCA", "提醒机器人",
		"改 DCA", "改DCA", "list_dca", "我的 bot", "我的Bot", "创建 bot", "创建bot",
	}
	dcaGridTokens = []string{
		"定投", "dca", "DCA", "网格策略", "网格参数", "网格回测",
		"generate_dca", "generate_grid", "loopback",
	}
	backtestHistoryTokens = []string{"历史回测", "上次结果", "上次回测", "回测记录", "回测历史"}
)
