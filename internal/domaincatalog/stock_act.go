package domaincatalog

import "strings"

// Stock analysis act values (finer than catalog plugin Act=analyze).
const (
	StockActAnalyze           = "analyze"
	StockActQuotePrice        = "quote_price"
	StockActTechnicalAnalysis = "technical_analysis"
	StockActContextFollowup   = "context_followup"
	StockActSymbolResolve     = "symbol_resolve"
	StockActMultiSymbol       = "multi_symbol_delegate"
)

// NormalizeStockAct returns a canonical stock act or analyze when unknown/empty.
func NormalizeStockAct(act string) string {
	switch strings.TrimSpace(act) {
	case StockActQuotePrice, StockActTechnicalAnalysis, StockActContextFollowup, StockActSymbolResolve, StockActMultiSymbol:
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
		return ProfileStockPriceSnapshot
	case StockActTechnicalAnalysis:
		return ProfileStockTechnicalFull
	case StockActContextFollowup:
		return ProfileStockContextFollowup
	case StockActSymbolResolve:
		return ProfileStockSymbolResolve
	case StockActMultiSymbol:
		return ProfileSubagentMultiStock
	default:
		return ""
	}
}

// ProfileExecutionHint returns a short ReAct hint for the loop prompt.
func ProfileExecutionHint(profileID string) string {
	switch profileID {
	case ProfileStockPriceSnapshot:
		return "查现价：search_code 后调用 get_current_price 返回现价即可；不要走 get_mcp_analysis。"
	case ProfileStockPriceViaMCP:
		return "查股价须 search_code 后走 get_single_prompt_template(tag=price) + get_mcp_analysis；不要仅用 get_current_price 收尾。"
	case ProfileStockTechnicalFull:
		return "技术面/K线须 get_single_prompt_template + get_mcp_analysis；标的已在上下文时可省略重复 search_code。"
	case ProfileStockContextFollowup:
		return "续问同一标的时复用已解析 code；需要走势/技术面时优先 get_mcp_analysis。"
	case ProfileStockSymbolResolve:
		return "切换标的时必须在本轮重新 search_code 确认新代码。"
	case ProfileSubagentMultiStock:
		return "本回合你是编排者（类似 Cursor Task）：必须调用 delegate_tasks 一次，tasks[] 每路一个标的、并行执行；子 Agent 自带独立上下文。主回合禁止 search_code/get_mcp_analysis/get_current_price；等 delegate_tasks 返回后再汇总对比答复。"
	default:
		return ""
	}
}
