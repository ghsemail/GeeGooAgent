package strategydev

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// applyDailyMacroKlineDisplayForEmit rewrites K-line pattern UI for macro monitor:
// only the daily aligned column (no 60m/weekly grid, no MTF weight_schema summary).
func applyDailyMacroKlineDisplayForEmit(out *LabEpisodeDetail) {
	if out == nil || strings.TrimSpace(out.KlineJEVProfile) != LabKlineJEVProfileDailyMacro {
		return
	}
	applyDailyMacroNonKlineDisplayForEmit(out)
	daily := labDetailDailyMTFEntry(out)
	if daily == nil {
		daily = &slots.KlinePatternMTFEntry{
			Frequency: "daily",
			Label:     "日线",
			AlignNote: "日线未对齐",
		}
	}
	ent := *daily
	if strings.TrimSpace(ent.Frequency) == "" {
		ent.Frequency = "daily"
	}
	if strings.TrimSpace(ent.Label) == "" {
		ent.Label = "日线"
	}
	threeBar := ent.ThreeBar
	hits := labJEVHitsFromThreeBarAnchor(threeBar)
	summary := slots.KlinePatternSummary(hits)
	if summary == "" && labJEVThreeBarHasAnchorOHLC(threeBar) {
		summary = labJEVKlineNoNamedPatternSummary
	}
	bias := ent.Bias
	if bias == nil && threeBar != nil {
		bias = labJEVBiasFromThreeBar(threeBar)
	}
	mtf := &slots.KlinePatternMTFContext{
		SignalFrequency: "daily",
		Timeframes:      []slots.KlinePatternMTFEntry{ent},
	}
	if out.KlinePattern == nil {
		out.KlinePattern = &LabKlinePatternDetail{}
	}
	out.KlinePattern.Summary = summary
	out.KlinePattern.Bias = bias
	out.KlinePattern.Patterns = hits
	out.KlinePattern.ThreeBar = threeBar
	out.KlinePattern.MTF = mtf
	applyLabKlinePatternTopLevel(out)
}

func applyDailyMacroNonKlineDisplayForEmit(out *LabEpisodeDetail) {
	if out.PriceAnalysis != nil {
		var rows []PriceAnalysisRow
		for _, row := range out.PriceAnalysis.Rows {
			if rowLooksDailyTF(row.Timeframe) {
				rows = append(rows, row)
			}
		}
		out.PriceAnalysis.Rows = rows
		out.PriceAnalysis.FilterTimeframe = "日线"
	}
	if out.TechnicalAnalysis != nil && len(out.TechnicalAnalysis.Timeframes) > 0 {
		out.TechnicalAnalysis.Timeframes = filterLabTFDailyOnly(out.TechnicalAnalysis.Timeframes)
	}
	if out.IndexTrend != nil && len(out.IndexTrend.Timeframes) > 0 {
		out.IndexTrend.Timeframes = filterLabIndexTrendDailyOnly(out.IndexTrend.Timeframes)
	}
}

func rowLooksDailyTF(tf string) bool {
	switch NormalizeFrequency(tf) {
	case "daily":
		return true
	default:
		return strings.TrimSpace(tf) == "日线"
	}
}

func filterLabTFDailyOnly(m map[string]LabTechnicalAnalysisTF) map[string]LabTechnicalAnalysisTF {
	out := make(map[string]LabTechnicalAnalysisTF, 1)
	for k, v := range m {
		if NormalizeFrequency(k) == "daily" || NormalizeFrequency(v.Frequency) == "daily" {
			out["daily"] = v
			break
		}
	}
	return out
}

func filterLabIndexTrendDailyOnly(m map[string]LabIndexTrendTF) map[string]LabIndexTrendTF {
	out := make(map[string]LabIndexTrendTF, 1)
	for k, v := range m {
		if NormalizeFrequency(k) == "daily" || NormalizeFrequency(v.Frequency) == "daily" {
			out["daily"] = v
			break
		}
	}
	return out
}

func labDetailDailyMTFEntry(out *LabEpisodeDetail) *slots.KlinePatternMTFEntry {
	if e := labJEVMTFEntry(out, "daily"); e != nil {
		return e
	}
	if out.KlinePattern != nil && out.KlinePattern.MTF != nil {
		want := NormalizeFrequency("daily")
		for i := range out.KlinePattern.MTF.Timeframes {
			tf := &out.KlinePattern.MTF.Timeframes[i]
			if NormalizeFrequency(tf.Frequency) == want {
				return tf
			}
		}
	}
	return nil
}
