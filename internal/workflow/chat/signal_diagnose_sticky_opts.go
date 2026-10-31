package chat

import (
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

// MergeSignalDiagnoseOpts overlays incoming onto base; incoming non-empty fields win.
func MergeSignalDiagnoseOpts(base, incoming *runtime.SignalDiagnoseOpts) *runtime.SignalDiagnoseOpts {
	return runtime.MergeSignalDiagnoseOpts(base, incoming)
}

func applySignalDiagnoseOptsToFlow(o *runtime.SignalDiagnoseOpts, flow *Flow) {
	if o == nil || flow == nil {
		return
	}
	if o.UsePriceAnalysisEpisodeFilter != nil {
		flow.UsePriceAnalysisEpisodeFilter = *o.UsePriceAnalysisEpisodeFilter
	}
	if o.PriceAnalysisScoreThreshold != nil {
		flow.PriceAnalysisScoreThreshold = *o.PriceAnalysisScoreThreshold
	}
	if o.PriceAnalysisBuyScoreDirection != "" {
		flow.PriceAnalysisBuyScoreDirection = strategydevParseBuyDir(o.PriceAnalysisBuyScoreDirection)
	}
	if o.PriceAnalysisGateScoreMode != "" {
		flow.PriceAnalysisGateScoreMode = strategydevParseGateMode(o.PriceAnalysisGateScoreMode)
	}
	if len(o.BuySignal) > 0 {
		flow.ProbeBuyOverride = append([]any(nil), o.BuySignal...)
	}
	if len(o.SellSignal) > 0 {
		flow.ProbeSellOverride = append([]any(nil), o.SellSignal...)
	}
	if o.SellMerge != "" {
		flow.ProbeSellMerge = o.SellMerge
	}
	if o.BuyMerge != "" {
		flow.ProbeBuyMerge = o.BuyMerge
	}
	if o.Frequency != "" {
		flow.ProbeFrequencyOverride = o.Frequency
	}
	if o.MonthsBack > 0 {
		flow.MonthsBack = o.MonthsBack
	}
	if o.PanelStrategyLabel != "" {
		flow.ProbePanelLabel = o.PanelStrategyLabel
	}
	if o.TradeDirection != nil {
		flow.TradeDirection = *o.TradeDirection
	}
	if o.LabMode != nil {
		flow.LabMode = *o.LabMode
	}
}

func strategydevParseBuyDir(s string) strategydev.BuyScoreDirection {
	return effectiveSignalDiagnoseBuyScoreDirection(strategydev.ParseBuyScoreDirection(s))
}

func strategydevParseGateMode(s string) strategydev.PriceAnalysisGateScoreMode {
	return strategydev.EffectiveGateScoreMode(strategydev.ParsePriceAnalysisGateScoreMode(s))
}
