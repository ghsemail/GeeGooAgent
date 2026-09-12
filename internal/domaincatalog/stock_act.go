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
	return ExecutionProfileForDomain(domain, act)
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
		return "多标的并行（Cursor Task 风格）：优先 delegate_tasks 一次（tasks[] 每标的一项）。返回后必须在最终回复中逐标的给出具体现价/分析，并做简要对比；禁止只描述委派过程而不写数据。"
	case ProfileSignalProbeExecute:
		return "信号测试：clarify 选完策略名后，本轮必须调用 probe_bot_signal_series（沿用会话 code、months_back=3）；禁止只确认选择而不 probe。"
	case ProfileSignalProbeSymbolSwitch:
		return "换标的：必须先 search_code 确认新 code，再 probe_bot_signal_series；禁止沿用上一轮 NVDA/其他 ticker。"
	default:
		return ""
	}
}
