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

// NormalizeStockAct returns a canonical stock act or analyze when unknown/empty.
func NormalizeStockAct(act string) string {
	switch strings.TrimSpace(act) {
	case StockActQuotePrice, StockActTechnicalAnalysis, StockActContextFollowup, StockActSymbolResolve:
		return strings.TrimSpace(act)
	default:
		return StockActAnalyze
	}
}

// ExecutionProfileFor returns the profile id for a domain+act pair, or "" if none.
func ExecutionProfileFor(domain Domain, act string) string {
	if domain != DomainStockAnalysis {
		return ""
	}
	switch NormalizeStockAct(act) {
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
