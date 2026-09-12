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

