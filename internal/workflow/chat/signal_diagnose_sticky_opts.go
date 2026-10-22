package chat

import (
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

// MergeSignalDiagnoseOpts overlays incoming onto base; incoming non-empty fields win.
func MergeSignalDiagnoseOpts(base, incoming *runtime.SignalDiagnoseOpts) *runtime.SignalDiagnoseOpts {
	if incoming == nil {
		return cloneSignalDiagnoseOpts(base)
	}
	if base == nil {
		return cloneSignalDiagnoseOpts(incoming)
	}
	out := cloneSignalDiagnoseOpts(base)
	if incoming.UsePriceAnalysisEpisodeFilter != nil {
		out.UsePriceAnalysisEpisodeFilter = incoming.UsePriceAnalysisEpisodeFilter
	}
	if incoming.PriceAnalysisScoreThreshold != nil {
		out.PriceAnalysisScoreThreshold = incoming.PriceAnalysisScoreThreshold
	}
	if incoming.PriceAnalysisBuyScoreDirection != "" {
		out.PriceAnalysisBuyScoreDirection = incoming.PriceAnalysisBuyScoreDirection
	}
	if incoming.PriceAnalysisGateScoreMode != "" {
		out.PriceAnalysisGateScoreMode = incoming.PriceAnalysisGateScoreMode
	}
	if len(incoming.BuySignal) > 0 {
		out.BuySignal = cloneAnySlice(incoming.BuySignal)
	}
	if len(incoming.SellSignal) > 0 {
		out.SellSignal = cloneAnySlice(incoming.SellSignal)
	}
	if incoming.SellMerge != "" {
		out.SellMerge = incoming.SellMerge
	}
	if incoming.BuyMerge != "" {
		out.BuyMerge = incoming.BuyMerge
	}
	if incoming.Frequency != "" {
		out.Frequency = incoming.Frequency
	}
	if incoming.MonthsBack > 0 {
		out.MonthsBack = incoming.MonthsBack
	}
	if incoming.PanelStrategyLabel != "" {
		out.PanelStrategyLabel = incoming.PanelStrategyLabel
	}
	if incoming.TradeDirection != nil {
		td := *incoming.TradeDirection
		out.TradeDirection = &td
	}
	if incoming.LabMode != nil {
		out.LabMode = incoming.LabMode
	}
	return out
}

func cloneSignalDiagnoseOpts(o *runtime.SignalDiagnoseOpts) *runtime.SignalDiagnoseOpts {
	if o == nil {
		return nil
	}
	out := *o
	if o.UsePriceAnalysisEpisodeFilter != nil {
		v := *o.UsePriceAnalysisEpisodeFilter
		out.UsePriceAnalysisEpisodeFilter = &v
	}
	if o.PriceAnalysisScoreThreshold != nil {
		v := *o.PriceAnalysisScoreThreshold
		out.PriceAnalysisScoreThreshold = &v
	}
	if o.TradeDirection != nil {
		td := *o.TradeDirection
		out.TradeDirection = &td
	}
	if len(o.BuySignal) > 0 {
		out.BuySignal = cloneAnySlice(o.BuySignal)
	}
	if len(o.SellSignal) > 0 {
		out.SellSignal = cloneAnySlice(o.SellSignal)
	}
	return &out
}

func cloneAnySlice(in []any) []any {
	if len(in) == 0 {
		return nil
	}
	out := make([]any, len(in))
	copy(out, in)
	return out
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
		flow.ProbeBuyOverride = cloneAnySlice(o.BuySignal)
	}
	if len(o.SellSignal) > 0 {
		flow.ProbeSellOverride = cloneAnySlice(o.SellSignal)
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
