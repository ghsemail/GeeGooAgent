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
		for _, d := range details {
			ref := LabDetailRef(side, d.StartIdx)
			sum := LabEpisodeCardSummary{
				DetailRef: ref,
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
