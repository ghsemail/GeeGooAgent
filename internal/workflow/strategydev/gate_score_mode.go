package strategydev

import "strings"

// PriceAnalysisGateScoreMode selects which multi-TF composite is used for buy intercept (soft/hard score gate).
// JSON/API: price_analysis_gate_score_mode = "full" | "htf_only" | "tf_tiered" (default full).
type PriceAnalysisGateScoreMode string

const (
	// PriceAnalysisGateScoreModeFull: gate score equals display price_score (weekly + daily + entry TF).
	PriceAnalysisGateScoreModeFull PriceAnalysisGateScoreMode = "full"
	// PriceAnalysisGateScoreModeHTFOnly: gate uses only TFs strictly coarser than the signal/entry period.
	PriceAnalysisGateScoreModeHTFOnly PriceAnalysisGateScoreMode = "htf_only"
	// PriceAnalysisGateScoreModeTFTiered: gate uses fixed TF weights by signal period (60m/5m); daily/weekly fall back to htf_only.
	PriceAnalysisGateScoreModeTFTiered PriceAnalysisGateScoreMode = "tf_tiered"
)

// ParsePriceAnalysisGateScoreMode parses workflow/diagnose option (unknown → full).
func ParsePriceAnalysisGateScoreMode(raw string) PriceAnalysisGateScoreMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(PriceAnalysisGateScoreModeHTFOnly), "htf", "htf-only":
		return PriceAnalysisGateScoreModeHTFOnly
	case string(PriceAnalysisGateScoreModeTFTiered), "tf-tiered", "tiered":
		return PriceAnalysisGateScoreModeTFTiered
	default:
		return PriceAnalysisGateScoreModeFull
	}
}

// EffectiveGateScoreMode returns full when unset.
func EffectiveGateScoreMode(m PriceAnalysisGateScoreMode) PriceAnalysisGateScoreMode {
	switch m {
	case PriceAnalysisGateScoreModeHTFOnly:
		return PriceAnalysisGateScoreModeHTFOnly
	case PriceAnalysisGateScoreModeTFTiered:
		return PriceAnalysisGateScoreModeTFTiered
	default:
		return PriceAnalysisGateScoreModeFull
	}
}

func gateScoreModeDisplayLabel(m PriceAnalysisGateScoreMode) string {
	switch m {
	case PriceAnalysisGateScoreModeHTFOnly:
		return "递推（仅更大周期）"
	case PriceAnalysisGateScoreModeTFTiered:
		return "按信号周期分档权重"
	default:
		return "全周期"
	}
}
