package strategydev

import "strings"

// PriceAnalysisGateScoreMode selects which multi-TF composite is used for buy intercept (soft/hard score gate).
// JSON/API: price_analysis_gate_score_mode = "full" | "htf_only" (default full).
type PriceAnalysisGateScoreMode string

const (
	// PriceAnalysisGateScoreModeFull: gate score equals display price_score (weekly + daily + entry TF).
	PriceAnalysisGateScoreModeFull PriceAnalysisGateScoreMode = "full"
	// PriceAnalysisGateScoreModeHTFOnly: gate uses only TFs strictly coarser than the signal/entry period.
	PriceAnalysisGateScoreModeHTFOnly PriceAnalysisGateScoreMode = "htf_only"
)

// ParsePriceAnalysisGateScoreMode parses workflow/diagnose option (unknown → full).
func ParsePriceAnalysisGateScoreMode(raw string) PriceAnalysisGateScoreMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(PriceAnalysisGateScoreModeHTFOnly), "htf", "htf-only":
		return PriceAnalysisGateScoreModeHTFOnly
	default:
		return PriceAnalysisGateScoreModeFull
	}
}

// EffectiveGateScoreMode returns full when unset.
func EffectiveGateScoreMode(m PriceAnalysisGateScoreMode) PriceAnalysisGateScoreMode {
	if m == PriceAnalysisGateScoreModeHTFOnly {
		return PriceAnalysisGateScoreModeHTFOnly
	}
	return PriceAnalysisGateScoreModeFull
}

func gateScoreModeDisplayLabel(m PriceAnalysisGateScoreMode) string {
	if m == PriceAnalysisGateScoreModeHTFOnly {
		return "递推（仅更大周期）"
	}
	return "全周期"
}
