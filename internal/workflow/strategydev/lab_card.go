package strategydev

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// LabCardEpisodeSummaries maps detail_ref → card summaries for SSE payload.
type LabCardEpisodeSummaries map[string]LabEpisodeCardSummary

// BuildLabCardSummariesFromEval builds lightweight lab lines for each episode start bar.
func BuildLabCardSummariesFromEval(eval slots.SignalEpisodeEval, patternHits []any, flowPriceSummary string, longOnly bool) LabCardEpisodeSummaries {
	out := LabCardEpisodeSummaries{}
	appendSide := func(side string, details []slots.SignalEpisodeDetail) {
		for epIdx, d := range details {
			ref := LabDetailRef(side, d.StartIdx)
			epIdxCopy := epIdx
			sum := LabEpisodeCardSummary{
				DetailRef:    ref,
				StartIdx:     d.StartIdx,
				EpisodeIndex: &epIdxCopy,
				KlinePattern: LightKlineSummaryAtIndex(patternHits, d.StartIdx),
			}
			if flowPriceSummary != "" {
				sum.PriceAnalysis = flowPriceSummary
			}
			out[ref] = sum
		}
	}
	appendSide("buy", eval.BuyDetails)
	if !longOnly {
		appendSide("sell", eval.SellDetails)
	}
	return out
}

// BuildLabCardSummariesFromMerged adds detail_ref rows for each buy/sell trigger bar (episode-independent).
// When longOnly is true, sell-side trigger rows are omitted (matches signal_eval long_only cohorts).
func BuildLabCardSummariesFromMerged(bars, buyMerged, sellMerged, patternHits []any, flowPriceSummary string, longOnly bool) LabCardEpisodeSummaries {
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
	if !longOnly {
		add("sell", sellMerged, -1)
	}
	return out
}

// FilterLabCardSummariesForLongOnly drops sell:* refs from lab episode lists (Persist / Monday lab panel).
func FilterLabCardSummariesForLongOnly(sums LabCardEpisodeSummaries, longOnly bool) LabCardEpisodeSummaries {
	if !longOnly || len(sums) == 0 {
		return sums
	}
	out := LabCardEpisodeSummaries{}
	for k, v := range sums {
		if strings.HasPrefix(k, "sell:") {
			continue
		}
		out[k] = v
	}
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
