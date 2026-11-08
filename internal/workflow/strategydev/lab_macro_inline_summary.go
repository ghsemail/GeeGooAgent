package strategydev

import (
	"strings"
)

// LabEpisodeSummaryForMacro builds FE-compatible lab_episode_summary for macro inline Mongo.
func LabEpisodeSummaryForMacro(detail LabEpisodeDetail) map[string]any {
	ref := strings.TrimSpace(detail.Ref)
	if ref == "" {
		return nil
	}
	klineLine := strings.TrimSpace(detail.KlinePatternSummary)
	if klineLine == "" && detail.KlinePattern != nil {
		klineLine = strings.TrimSpace(detail.KlinePattern.Summary)
	}
	if klineLine == "" && detail.KlineJEV != nil {
		if r := strings.TrimSpace(detail.KlineJEV.RecommendedReason); r != "" {
			klineLine = r
		} else if c := strings.TrimSpace(detail.KlineJEV.RecommendedChoice); c != "" {
			klineLine = c
		}
	}
	if klineLine != "" {
		klineLine = LabCardSummaryLine("kline", klineLine)
	}
	priceLine := ""
	if detail.PriceAnalysis != nil {
		priceLine = strings.TrimSpace(detail.PriceAnalysis.Summary)
		if priceLine == "" {
			for _, row := range detail.PriceAnalysis.Rows {
				line := strings.TrimSpace(row.Conclusion)
				if line == "" {
					continue
				}
				priceLine = line
				break
			}
		}
	}
	if priceLine != "" {
		priceLine = LabCardSummaryLine("price", priceLine)
	}
	out := map[string]any{
		"detail_ref": ref,
	}
	if klineLine != "" {
		out["kline_pattern"] = klineLine
	}
	if priceLine != "" {
		out["price_analysis"] = priceLine
	}
	if detail.PriceAnalysisJEV != nil {
		out["price_analysis_jev"] = detail.PriceAnalysisJEV
	}
	if detail.KlineJEV != nil {
		out["kline_jev"] = detail.KlineJEV
	}
	if detail.TechnicalAnalysisJEV != nil {
		out["technical_analysis_jev"] = detail.TechnicalAnalysisJEV
	}
	if detail.IndexTrendJEV != nil {
		out["index_trend_jev"] = detail.IndexTrendJEV
	}
	if detail.JEVUnified != nil {
		out["jev_unified"] = detail.JEVUnified
	}
	if len(out) <= 1 {
		return nil
	}
	return out
}
