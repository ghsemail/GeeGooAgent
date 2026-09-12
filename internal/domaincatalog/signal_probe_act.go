package domaincatalog

import "github.com/ghsemail/GeeGooAgent/internal/slots"

// ExecutionProfileForDomain returns the profile id for a domain+act pair, or "" if none.
func ExecutionProfileForDomain(domain Domain, act string) string {
	return ProbeExecutionProfile(domain, act, "")
}

// ProbeExecutionProfile picks signal_probe contracts; userText enables symbol-switch detection.
func ProbeExecutionProfile(domain Domain, act string, userText string) string {
	if id := executionProfileForStock(domain, act); id != "" {
		return id
	}
	if domain == DomainSignalProbe {
		if slots.ShouldResolveNewSymbolForProbe(userText) {
			return ProfileSignalProbeSymbolSwitch
		}
		return ProfileSignalProbeExecute
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
