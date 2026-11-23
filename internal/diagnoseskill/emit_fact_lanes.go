package diagnoseskill

import (
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

type anchorBarOHLC struct {
	open, high, low float64
}

func anchorOHLCFromBundle(bundle LaneBundle, macro bool, signalFreq string) *anchorBarOHLC {
	anchorFreq := signalFreq
	if macro {
		anchorFreq = "daily"
	}
	klineRows, _ := laneArray(bundle.KlinePattern, "kline_pattern")
	paRows, _ := laneArray(bundle.PriceAnalysis, "price_analysis")
	for _, rows := range [][]map[string]any{klineRows, paRows} {
		for _, row := range rows {
			f, _ := row["frequency"].(string)
			if strategydev.NormalizeFrequency(f) != strategydev.NormalizeFrequency(anchorFreq) {
				continue
			}
			o, _ := row["open"].(float64)
			h, _ := row["high"].(float64)
			l, _ := row["low"].(float64)
			if o > 0 || h > 0 || l > 0 {
				return &anchorBarOHLC{open: o, high: h, low: l}
			}
		}
	}
	return nil
}

// applySkillFactLanes writes lab-detail-shaped fact payloads into skill `result`
// (replaces raw macro-api lane arrays so trading_operation can render PA/TA/kline/IT tables).
func applySkillFactLanes(res Result, detail *strategydev.LabEpisodeDetail) {
	if res == nil || detail == nil {
		return
	}
	klineMap := skillKlinePatternMap(detail)
	strategydev.PrepareLabEpisodeDetailForEmit(detail)

	if detail.PriceAnalysis != nil {
		pa := *detail.PriceAnalysis
		pa.PriceAnalysisJEV = nil
		res["price_analysis"] = pa
	} else {
		res["price_analysis"] = map[string]any{}
	}

	if detail.TechnicalAnalysis != nil {
		res["technical_analysis"] = detail.TechnicalAnalysis
	} else {
		res["technical_analysis"] = map[string]any{}
	}

	if detail.IndexTrend != nil {
		res["index_trend"] = detail.IndexTrend
	} else {
		res["index_trend"] = map[string]any{}
	}

	if len(klineMap) > 0 {
		res["kline_pattern"] = klineMap
	} else {
		res["kline_pattern"] = map[string]any{}
	}

	if p := detail.KlineJEVProfile; p != "" {
		res["kline_jev_profile"] = p
	}
}

func skillKlinePatternMap(detail *strategydev.LabEpisodeDetail) map[string]any {
	out := map[string]any{}
	if detail.KlinePattern != nil {
		if s := detail.KlinePattern.Summary; s != "" {
			out["summary"] = s
		}
		if len(detail.KlinePattern.Patterns) > 0 {
			out["patterns"] = detail.KlinePattern.Patterns
			out["kline_patterns"] = detail.KlinePattern.Patterns
		}
		if detail.KlinePattern.ThreeBar != nil {
			out["three_bar"] = detail.KlinePattern.ThreeBar
			out["kline_pattern_three_bar"] = detail.KlinePattern.ThreeBar
		}
		if detail.KlinePatternSummary != "" {
			out["kline_pattern_summary"] = detail.KlinePatternSummary
		}
	}
	if len(detail.KlinePatterns) > 0 && out["kline_patterns"] == nil {
		out["kline_patterns"] = detail.KlinePatterns
	}
	if detail.KlinePatternThreeBar != nil && out["three_bar"] == nil {
		out["kline_pattern_three_bar"] = detail.KlinePatternThreeBar
	}
	if detail.KlinePatternSummary != "" && out["summary"] == nil {
		out["summary"] = detail.KlinePatternSummary
	}
	return out
}