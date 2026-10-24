package strategydev

import (
	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// LabCardEpisodeSummaries maps detail_ref → card summaries for SSE payload.
type LabCardEpisodeSummaries map[string]LabEpisodeCardSummary

// BuildLabCardSummariesFromEval builds lightweight lab lines for each episode start bar.
func BuildLabCardSummariesFromEval(eval slots.SignalEpisodeEval, patternHits []any, flowPriceSummary string) LabCardEpisodeSummaries {
	out := LabCardEpisodeSummaries{}
	appendSide := func(side string, details []slots.SignalEpisodeDetail) {
		for epIdx, d := range details {
			ref := LabDetailRef(side, d.StartIdx)
			sum := LabEpisodeCardSummary{
				DetailRef:    ref,
				StartIdx:     d.StartIdx,
				EpisodeIndex: epIdx,
				KlinePattern: LightKlineSummaryAtIndex(patternHits, d.StartIdx),
			}
			if flowPriceSummary != "" {
				sum.PriceAnalysis = flowPriceSummary
			}
			out[ref] = sum
		}
	}
	appendSide("buy", eval.BuyDetails)
	appendSide("sell", eval.SellDetails)
	return out
}

// BuildLabCardSummariesFromMerged adds detail_ref rows for each buy/sell trigger bar (episode-independent).
func BuildLabCardSummariesFromMerged(bars, buyMerged, sellMerged, patternHits []any, flowPriceSummary string) LabCardEpisodeSummaries {
	out := LabCardEpisodeSummaries{}
	n := len(bars)
	if n == 0 {
		return out
	}
	add := func(side string, merged []any, target int) {
		for i := 0; i < n && i < len(merged); i++ {
			if int(slotsFloat(merged[i])) != target {
				continue
			}
			ref := LabDetailRef(side, i)
			out[ref] = LabEpisodeCardSummary{
				DetailRef:     ref,
				StartIdx:      i,
				KlinePattern:  LightKlineSummaryAtIndex(patternHits, i),
				PriceAnalysis: flowPriceSummary,
			}
		}
	}
	add("buy", buyMerged, 1)
	add("sell", sellMerged, -1)
	return out
}

func slotsFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	default:
		return 0
	}
}

// MergeLabCardSummaries merges trigger-level summaries; eval rows win on duplicate refs.
func MergeLabCardSummaries(parts ...LabCardEpisodeSummaries) LabCardEpisodeSummaries {
	out := LabCardEpisodeSummaries{}
	for _, p := range parts {
		for k, v := range p {
			if _, ok := out[k]; !ok {
				out[k] = v
			}
		}
	}
	return out
}
