package cognition

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// RulePlanner classifies turns with domain rules. Gray-zone utterances
// become clarify/talk instead of silently executing a playbook.
type RulePlanner struct{}

// Plan implements Planner.
func (RulePlanner) Plan(in PlanInput) TurnPlan {
	msg := strings.TrimSpace(in.UserText)
	if in.LastDomain == DomainAmbiguous {
		if d, ok := mapClarifyChoice(msg); ok {
			p := planForDomain(d)
			p.Reason = "用户选择了上一轮澄清选项"
			p.Confidence = 0.9
			return p
		}
	}
	switch {
	case isDefinitionOrChitchat(msg):
		return planForDomain(DomainChat)
	case hasAny(msg, promptAdminTokens):
		return planForDomain(DomainPromptAdmin)
	case hasAny(msg, customSignalTokens):
		return planForDomain(DomainCustomSignal)
	case hasAny(msg, reportWriteTokens):
		return planForDomain(DomainReportWrite)
	case hasAny(msg, reportLookupTokens):
		return planForDomain(DomainReportLookup)
	case hasAny(msg, knowledgeTokens):
		return planForDomain(DomainKnowledge)
	case hasAny(msg, newsTokens):
		return planForDomain(DomainNews)
	case hasAny(msg, botManageTokens):
		p := planForDomain(DomainBotManage)
		if hasAny(msg, []string{"建", "创建", "改", "删除", "停"}) {
			p.Mode = ModeExecute
		}
		return p
	case hasAny(msg, dcaGridTokens):
		return planForDomain(DomainDCAGrid)
	case hasAny(msg, backtestHistoryTokens):
		return planForDomain(DomainBacktestHistory)
	case isCompoundIntent(msg):
		return planForDomain(DomainAmbiguous)
	case isSignalProbe(msg):
		return planForDomain(DomainSignalProbe)
	case isBacktestRun(msg):
		return planForDomain(DomainBacktestRun)
	case isStickyDomain(in.LastDomain) && (isFollowUpUtterance(msg) || isBareStrategyTalk(msg) || slots.IsLikelyStockUtterance(msg)):
		p := planForDomain(in.LastDomain)
		p.Reason = "沿用上一轮领域 " + string(in.LastDomain)
		p.Confidence = 0.8
		return p
	case isBareStrategyTalk(msg):
		return planForDomain(DomainAmbiguous)
	case isStockAnalysis(msg):
		return planForDomain(DomainStockAnalysis)
	case slots.IsLikelyStockUtterance(msg):
		p := planForDomain(DomainStockAnalysis)
		p.Reason = "口语标的名，默认个股分析"
		p.Confidence = 0.82
		return p
	default:
		return planForDomain(DomainChat)
	}
}

func isStickyDomain(d Domain) bool {
	return d != "" && d != DomainChat && d != DomainAmbiguous
}

func isFollowUpUtterance(msg string) bool {
	return hasAny(msg, []string{"换成", "改成", "继续", "再看看", "还是那个", "同样的", "刚才那个标的", "换一个标的"})
}

func mapClarifyChoice(msg string) (Domain, bool) {
	switch {
	case hasAny(msg, []string{"个股/指标分析", "指标分析", "先分析"}):
		return DomainStockAnalysis, true
	case msg == "分析":
		return DomainStockAnalysis, true
	case hasAny(msg, []string{"测买卖点", "只测点"}):
		return DomainSignalProbe, true
	case hasAny(msg, []string{"跑回测看收益", "看收益"}):
		return DomainBacktestRun, true
	case hasAny(msg, []string{"先问答", "先不操作"}):
		return DomainChat, true
	default:
		return "", false
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

func isCompoundIntent(msg string) bool {
	verbs := 0
	if hasAny(msg, []string{
		"分析", "技术面", "基本面", "走势", "趋势", "行情", "多少钱", "股价",
	}) {
		verbs++
	}
	if hasAny(msg, []string{"回测", "跑回测", "再回测", "backtest", "看收益"}) {
		verbs++
	}
	if hasAny(msg, []string{"测信号", "买卖点", "有没有买卖", "信号密度", "只看买卖"}) {
		verbs++
	}
	return verbs >= 2
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
