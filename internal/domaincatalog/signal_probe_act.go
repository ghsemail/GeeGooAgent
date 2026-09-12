package domaincatalog

// ExecutionProfileForDomain returns the profile id for a domain+act pair, or "" if none.
func ExecutionProfileForDomain(domain Domain, act string) string {
	if id := executionProfileForStock(domain, act); id != "" {
		return id
	}
	return executionProfileForSignalProbe(domain, act)
}

func executionProfileForStock(domain Domain, act string) string {
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

func executionProfileForSignalProbe(domain Domain, act string) string {
	if domain != DomainSignalProbe {
		return ""
	}
	if act != "" && act != "probe" {
		return ""
	}
	return ProfileSignalProbeExecute
}
