package priceanalysis

import "math"

// RoundLabRatio3 rounds ratio/subscore floats for JSON output (internal math stays full precision).
func RoundLabRatio3(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return v
	}
	rounded := math.Round(v*1000) / 1000
	if rounded == 0 {
		return 0
	}
	return rounded
}

// RoundPriceAnalysisForOutput applies display rounding on lab PA response fields (not prices).
func RoundPriceAnalysisForOutput(pa PriceAnalysis) PriceAnalysis {
	out := pa
	if len(pa.Rows) > 0 {
		out.Rows = append([]PriceAnalysisRow(nil), pa.Rows...)
	}
	out.PriceScore = NormalizePriceScore(out.PriceScore)
	for i := range out.Rows {
		if out.Rows[i].ER != 0 {
			out.Rows[i].ER = RoundLabRatio3(out.Rows[i].ER)
		}
		if out.Rows[i].RVOL != 0 {
			out.Rows[i].RVOL = RoundLabRatio3(out.Rows[i].RVOL)
		}
	}
	if len(out.TFScores) > 0 {
		rounded := make(map[string]TFScoreBreakdown, len(out.TFScores))
		for k, br := range out.TFScores {
			br.Subscore = RoundLabRatio3(br.Subscore)
			br.Conclusion = RoundLabRatio3(br.Conclusion)
			br.Position = RoundLabRatio3(br.Position)
			br.BosChoch = RoundLabRatio3(br.BosChoch)
			rounded[k] = br
		}
		out.TFScores = rounded
	}
	return out
}
