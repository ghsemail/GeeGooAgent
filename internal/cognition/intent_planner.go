package cognition

import "strings"

func isStickyDomain(d Domain) bool {
	return d != "" && d != DomainChat && d != DomainAmbiguous
}

func isFollowUpUtterance(msg string) bool {
	return hasAny(msg, []string{"换成", "改成", "继续", "再看看", "再帮我", "接着", "还是那个", "同样的", "刚才那个", "刚才那次", "换一个标的", "换一个策略", "换策略", "它最近", "不聊"})
}

func isActiveTaskDomain(d Domain) bool {
	return d == DomainSignalProbe || d == DomainBacktestRun
}

func isQualityOpinion(msg string) bool {
	return hasAny(msg, []string{"靠谱吗", "准吗", "准确吗", "可靠吗", "有用吗", "怎么样"}) &&
		!hasAny(msg, []string{"走势", "股价", "行情", "K线", "技术面"})
}

func isExplicitTaskSwitch(_ TurnPlan, msg string) bool {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return false
	}
	if isBacktestRun(msg) {
		return true
	}
	if hasAny(msg, []string{"新闻", "Reminder", "SmartTrade", "知识库", "盘前", "报告写了"}) {
		return true
	}
	if hasAny(msg, []string{"有哪些", "列出", "看看我有哪些"}) &&
		hasAny(msg, []string{"策略", "Bot", "提醒", "信号策略", "组合信号"}) {
		return true
	}
	return hasAny(msg, []string{"分析", "走势", "股价", "查一下", "多少钱", "报价", "K线", "技术面"}) &&
		!isFollowUpUtterance(msg) && len([]rune(msg)) > 8
}

func mapClarifyChoice(msg string) (Domain, string, bool) {
	switch {
	case hasAny(msg, []string{"只要当前价", "查现价", "现价就行"}):
		return DomainStockAnalysis, "quote_price", true
	case hasAny(msg, []string{"分析价格走势", "走势分析", "做价格分析"}):
		return DomainStockAnalysis, "technical_analysis", true
	case hasAny(msg, []string{"个股/指标分析", "指标分析", "先分析", "先只做分析", "先做分析"}):
		return DomainStockAnalysis, "technical_analysis", true
	case msg == "分析":
		return DomainStockAnalysis, "technical_analysis", true
	case hasAny(msg, []string{"先只做回测", "先做回测"}):
		return DomainBacktestRun, "", true
	case hasAny(msg, []string{"测买卖点", "只测点"}):
		return DomainSignalProbe, "", true
	case hasAny(msg, []string{"跑回测看收益", "看收益"}):
		return DomainBacktestRun, "", true
	case hasAny(msg, []string{"先问答", "先不操作"}):
		return DomainChat, "", true
	case hasAny(msg, []string{"SAR信号搭配", "SAR信号配套", "SAR加MACD", "SAR+MACD", "MACD直方图"}):
		return DomainKnowledge, "", true
	case hasAny(msg, []string{"MACD金叉死叉", "金叉死叉"}):
		return DomainKnowledge, "", true
	default:
		return "", "", false
	}
}

func isBacktestRun(msg string) bool {
	if hasAny(msg, dcaGridTokens) {
		return false
	}
	if hasAny(msg, backtestHistoryTokens) {
		return false
	}
	if isCompoundBacktestRequest(msg) {
		return false
	}
	if isSignalProbe(msg) && !hasAny(msg, []string{"收益", "回撤", "成交笔", "pnl"}) {
		return false
	}
	return hasAny(msg, []string{"回测", "跑回测", "再回测", "backtest", "就用刚才那套", "再跑回测"})
}

func isCompoundBacktestRequest(msg string) bool {
	return hasAny(msg, []string{"分析", "看看", "解读", "讲讲"}) &&
		hasAny(msg, []string{"回测", "跑回测", "再跑回测"})
}

func isSignalProbe(msg string) bool {
	if hasAny(msg, []string{"测信号", "买卖点", "有没有买卖", "信号密度", "只看买卖"}) {
		return true
	}
	return strings.Contains(msg, "回测") && hasAny(msg, []string{"买卖", "有没有信号"}) &&
		!hasAny(msg, []string{"收益", "回撤", "成交笔"})
}

var (
	dcaGridTokens         = []string{
		"定投", "dca", "DCA", "网格策略", "网格参数", "网格回测",
		"generate_dca", "generate_grid", "loopback",
	}
	backtestHistoryTokens = []string{"历史回测", "上次结果", "上次回测", "回测记录", "回测历史"}
)

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

