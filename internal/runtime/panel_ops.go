package runtime

// MergeSignalDiagnoseOpts overlays incoming onto base; incoming non-empty fields win.
func MergeSignalDiagnoseOpts(base, incoming *SignalDiagnoseOpts) *SignalDiagnoseOpts {
	if incoming == nil {
		return CloneSignalDiagnoseOpts(base)
	}
	if base == nil {
		return CloneSignalDiagnoseOpts(incoming)
	}
	out := CloneSignalDiagnoseOpts(base)
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

// CloneSignalDiagnoseOpts deep-copies panel ops for session sticky storage.
func CloneSignalDiagnoseOpts(o *SignalDiagnoseOpts) *SignalDiagnoseOpts {
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

// MergedPanelOps returns sticky + inbound panel parameters for this turn.
func MergedPanelOps(session *Session) *SignalDiagnoseOpts {
	if session == nil {
		return nil
	}
	return MergeSignalDiagnoseOpts(session.StickyPanelOps, session.InboundPanelOps)
}

// ApplyPanelOpsToSignalProbeTool overrides probe/diagnose HTTP args from panel ops (skill path).
// Panel frequency and related fields win over LLM-supplied tool arguments.
// When inbound ops were present, they are merged into sticky (same as workflow Apply).
func ApplyPanelOpsToSignalProbeTool(session *Session, body map[string]any) {
	if session == nil || body == nil {
		return
	}
	hadInbound := session.InboundPanelOps != nil
	merged := MergedPanelOps(session)
	if merged == nil {
		return
	}
	if merged.Frequency != "" {
		body["frequency"] = merged.Frequency
	}
	if merged.MonthsBack > 0 {
		body["months_back"] = merged.MonthsBack
	}
	if len(merged.BuySignal) > 0 {
		body["buy_signal"] = cloneAnySlice(merged.BuySignal)
	}
	if len(merged.SellSignal) > 0 {
		body["sell_signal"] = cloneAnySlice(merged.SellSignal)
	}
	if merged.SellMerge != "" {
		body["sell_merge"] = merged.SellMerge
	}
	if merged.BuyMerge != "" {
		body["buy_merge"] = merged.BuyMerge
	}
	if hadInbound {
		session.StickyPanelOps = CloneSignalDiagnoseOpts(merged)
		session.InboundPanelOps = nil
	}
}
