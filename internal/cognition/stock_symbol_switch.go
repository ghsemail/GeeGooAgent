package cognition

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/domaincatalog"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// applyStockSymbolSwitchAct promotes gather/analyze turns to symbol_resolve when the user
// continues a stock_analysis session with an explicit new symbol (Plan-only LastDomain or
// full session), but the classifier returns generic analyze.
func applyStockSymbolSwitchAct(in PlanInput, plan TurnPlan) TurnPlan {
	if in.LastDomain != DomainStockAnalysis {
		return plan
	}
	if plan.Domain != DomainStockAnalysis || plan.Mode != ModeGather {
		return plan
	}
	switch domaincatalog.NormalizeStockAct(plan.Act) {
	case domaincatalog.StockActSymbolResolve,
		domaincatalog.StockActContextFollowup,
		domaincatalog.StockActMultiSymbol,
		domaincatalog.StockActQuotePrice,
		domaincatalog.StockActTechnicalAnalysis:
		return plan
	}
	if domaincatalog.NormalizeStockAct(plan.Act) != domaincatalog.StockActAnalyze {
		return plan
	}
	msg := strings.TrimSpace(in.UserText)
	if !shouldPromoteStockSymbolResolve(msg) {
		return plan
	}
	if slots.ExtractExplicitStockReference(msg) == "" {
		return plan
	}
	plan.Act = domaincatalog.StockActSymbolResolve
	if strings.TrimSpace(plan.Reason) != "" {
		plan.Reason = "stock session: explicit symbol switch → symbol_resolve (" + plan.Reason + ")"
	} else {
		plan.Reason = "stock session: explicit symbol switch → symbol_resolve"
	}
	return plan
}

func shouldPromoteStockSymbolResolve(msg string) bool {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return false
	}
	if isPronounStockFollowUp(msg) {
		return false
	}
	if !isFollowUpUtterance(msg) && !hasAny(msg, []string{"换成", "改看", "不看", "不聊", "换标的", "换一只", "换一支"}) {
		return false
	}
	return true
}

func isPronounStockFollowUp(msg string) bool {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return false
	}
	for _, prefix := range []string{"它", "这个", "那个", "这只", "那只", "该", "此"} {
		if strings.HasPrefix(msg, prefix) {
			return true
		}
	}
	return false
}
