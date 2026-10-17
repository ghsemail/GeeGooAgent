package strategydev

// SignalPriceAnalysisResponseKeys are optional GeeGooSignal probe/diagnose fields Agent must ignore when PA is off.
var SignalPriceAnalysisResponseKeys = []string{
	"price_analysis",
	"price_filter",
	"price_episode_filter_stats",
	"price_score",
	"gate_price_score",
	"price_analysis_buy_score_direction",
	"price_analysis_gate_score_mode",
	"use_price_analysis_episode_filter",
	"price_analysis_score_threshold",
}

// StripSignalPriceAnalysisPayload removes price-analysis fields from a Signal tool payload copy.
func StripSignalPriceAnalysisPayload(data map[string]any) {
	if data == nil || PriceAnalysisFeatureEnabled() {
		return
	}
	for _, key := range SignalPriceAnalysisResponseKeys {
		delete(data, key)
	}
}
