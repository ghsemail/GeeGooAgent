package agent

import (
	"github.com/ghsemail/GeeGooAgent/internal/cognition"
	"github.com/ghsemail/GeeGooAgent/internal/domaincatalog"
)

// cursorPlanSteps returns reference steps for dashboard / eval docs (not injected into RunTurn).
func cursorPlanSteps(plan cognition.TurnPlan) []string {
	switch plan.Domain {
	case cognition.DomainSignalProbe:
		return []string{
			"Confirm symbol and strategy/signal (clarify if missing)",
			"resolve_signal → probe_bot_signal_series",
			"Summarize buy/sell hits for the user",
		}
	case cognition.DomainBacktestRun:
		return []string{
			"Playbook: strategy-backtest-run (only backtest playbook; 回测 ≠ generate)",
			"Confirm symbol and signal (clarify if missing)",
			"run_strategy_backtest with resolved parameters",
			"Summarize PnL / log highlights",
		}
	case cognition.DomainBacktestHistory:
		return []string{"Query prior backtest logs", "Summarize relevant metrics"}
	case cognition.DomainStockAnalysis:
		switch domaincatalog.NormalizeStockAct(plan.Act) {
		case domaincatalog.StockActQuotePrice:
			return []string{"search_code (if needed)", "get_current_price", "Reply with quote snapshot"}
		case domaincatalog.StockActTechnicalAnalysis:
			return []string{"search_code (if needed)", "get_mcp_analysis", "Summarize trend/technicals"}
		case domaincatalog.StockActMultiSymbol:
			return []string{"delegate_tasks once with one task per symbol", "Merge results into a comparative reply"}
		case domaincatalog.StockActContextFollowup:
			return []string{"Reuse session symbol from WorkingState", "Continue analysis with appropriate stock tools"}
		case domaincatalog.StockActSymbolResolve:
			return []string{"search_code for the new symbol", "Continue the prior analysis intent on the new symbol"}
		default:
			return []string{"search_code (if needed)", "get_mcp_analysis or get_current_price as fit", "Answer the user's stock question"}
		}
	case cognition.DomainDCAGrid:
		if plan.Mode == cognition.ModeGather {
			return []string{"search_signals when user names a vague indicator (MACD/RSI)", "Present matches and clarify if 2+", "Help user pick a strategy"}
		}
		return []string{
			"User asked to generate/design a DCA or Grid plan (not ordinary 回测)",
			"get_signal_combinations → pick signal_id if needed",
			"generate_dca_strategy or generate_grid_strategy",
			"Optional loopback_strategy to validate the generated plan",
		}
	case cognition.DomainBotManage:
		return []string{"List or mutate bots/reminders via trading_bot tools", "Confirm outcome to user"}
	case cognition.DomainReportLookup:
		return []string{"Query report APIs", "Summarize report content"}
	case cognition.DomainKnowledge:
		return []string{"Search knowledge base", "Answer with citations"}
	case cognition.DomainAmbiguous:
		if plan.Mode == cognition.ModeClarify {
			return []string{"Call clarify to disambiguate intent before heavy tools", "Wait for user choice, then execute"}
		}
		return nil
	default:
		return nil
	}
}
