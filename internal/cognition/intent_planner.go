package cognition

import (
	"strings"
)

func isStickyDomain(d Domain) bool {
	return d != "" && d != DomainChat && d != DomainAmbiguous
}

func isFollowUpUtterance(msg string) bool {
	return hasAny(msg, []string{"换成", "改成", "继续", "再看看", "再帮我", "接着", "还是那个", "同样的", "刚才那个", "刚才那次", "换一个标的", "它最近", "不聊"})
}

func mapClarifyChoice(msg string) (Domain, string, bool) {
	switch {
	case hasAny(msg, []string{"只要当前价", "查现价", "现价就行"}):
		return DomainStockAnalysis, "quote_price", true
	case hasAny(msg, []string{"分析价格走势", "走势分析", "做价格分析"}):
		return DomainStockAnalysis, "technical_analysis", true
	case hasAny(msg, []string{"个股/指标分析", "指标分析", "先分析"}):
		return DomainStockAnalysis, "technical_analysis", true
	case msg == "分析":
		return DomainStockAnalysis, "technical_analysis", true
	case hasAny(msg, []string{"测买卖点", "只测点"}):
		return DomainSignalProbe, "", true
	case hasAny(msg, []string{"跑回测看收益", "看收益"}):
		return DomainBacktestRun, "", true
	case hasAny(msg, []string{"先问答", "先不操作"}):
		return DomainChat, "", true
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
	if isSignalProbe(msg) && !hasAny(msg, []string{"收益", "回撤", "成交笔", "pnl"}) {
		return false
	}
	return hasAny(msg, []string{"回测", "跑回测", "再回测", "backtest", "就用刚才那套", "再跑回测"})
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
