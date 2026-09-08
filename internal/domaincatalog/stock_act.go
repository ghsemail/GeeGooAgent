package domaincatalog

import "strings"

// Stock analysis act values (finer than catalog plugin Act=analyze).
const (
	StockActAnalyze           = "analyze"
	StockActQuotePrice        = "quote_price"
	StockActTechnicalAnalysis = "technical_analysis"
	StockActContextFollowup   = "context_followup"
	StockActSymbolResolve     = "symbol_resolve"
)

// RefineStockAnalysisAct maps user text + prior domain to a stock-analysis act.
func RefineStockAnalysisAct(userText string, lastDomain Domain) string {
	msg := strings.TrimSpace(userText)
	if msg == "" {
		return StockActAnalyze
	}
	if lastDomain == DomainStockAnalysis {
		if isStockSymbolSwitch(msg) {
			return StockActSymbolResolve
		}
		if isStockContextFollowup(msg) {
			return StockActContextFollowup
		}
		if isStockTechnicalAnalysis(msg) {
			return StockActTechnicalAnalysis
		}
	}
	if isStockQuotePrice(msg) {
		return StockActQuotePrice
	}
	if isStockTechnicalAnalysis(msg) {
		return StockActTechnicalAnalysis
	}
	return StockActAnalyze
}

// ExecutionProfileFor returns the profile id for a domain+act pair, or "" if none.
func ExecutionProfileFor(domain Domain, act string) string {
	if domain != DomainStockAnalysis {
		return ""
	}
	switch strings.TrimSpace(act) {
	case StockActQuotePrice:
		return ProfileStockPriceViaMCP
	case StockActTechnicalAnalysis:
		return ProfileStockTechnicalFull
	case StockActContextFollowup:
		return ProfileStockContextFollowup
	case StockActSymbolResolve:
		return ProfileStockSymbolResolve
	default:
		return ""
	}
}

// ProfileExecutionHint returns a short ReAct hint for the loop prompt.
func ProfileExecutionHint(profileID string) string {
	switch profileID {
	case ProfileStockPriceViaMCP:
		return "查股价须 search_code 后走 get_single_prompt_template(tag=price) + get_mcp_analysis；不要仅用 get_current_price 收尾。"
	case ProfileStockTechnicalFull:
		return "技术面/K线须 get_single_prompt_template + get_mcp_analysis；标的已在上下文时可省略重复 search_code。"
	case ProfileStockContextFollowup:
		return "续问同一标的时复用已解析 code；需要走势/技术面时优先 get_mcp_analysis。"
	case ProfileStockSymbolResolve:
		return "切换标的时必须在本轮重新 search_code 确认新代码。"
	default:
		return ""
	}
}

func isStockQuotePrice(msg string) bool {
	if hasToken(msg, []string{"技术面", "K线", "k线", "走势", "趋势", "MACD", "macd"}) {
		return false
	}
	return hasToken(msg, []string{
		"股价", "现价", "价格", "多少钱", "查一下", "涨跌", "行情",
	})
}

func isStockTechnicalAnalysis(msg string) bool {
	return hasToken(msg, []string{
		"技术面", "K线", "k线", "走势", "趋势", "形态", "指标",
	})
}

func isStockContextFollowup(msg string) bool {
	if !hasToken(msg, []string{"它", "这个", "那个", "继续", "再", "接着", "还是"}) {
		return false
	}
	return hasToken(msg, []string{"走势", "怎么样", "如何", "表现", "最近"})
}

func isStockSymbolSwitch(msg string) bool {
	return hasToken(msg, []string{
		"不聊", "换成", "改成", "换到", "切换", "改为", "不看", "别聊",
	})
}

func hasToken(msg string, tokens []string) bool {
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
