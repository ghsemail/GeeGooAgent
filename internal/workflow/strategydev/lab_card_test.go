package strategydev

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestBuildLabCardSummariesFromMergedTriggers(t *testing.T) {
	bars := []any{
		map[string]any{"close": 1.0},
		map[string]any{"close": 2.0},
		map[string]any{"close": 3.0},
	}
	buy := []any{0, 1, 0}
	sell := []any{0, 0, -1}
	out := BuildLabCardSummariesFromMerged(bars, buy, sell, nil, "")
	if len(out) != 2 {
		t.Fatalf("summaries=%d", len(out))
	}
	if _, ok := out[LabDetailRef("buy", 1)]; !ok {
		t.Fatal("missing buy:1")
	}
	if _, ok := out[LabDetailRef("sell", 2)]; !ok {
		t.Fatal("missing sell:2")
	}
}

func TestMergeLabCardSummariesEvalWins(t *testing.T) {
	eval := slots.SignalEpisodeEval{
		BuyDetails: []slots.SignalEpisodeDetail{{StartIdx: 1}},
	}
	fromEval := BuildLabCardSummariesFromEval(eval, nil, "flow-price")
	fromMerged := BuildLabCardSummariesFromMerged(
		[]any{map[string]any{}, map[string]any{}},
		[]any{0, 1},
		[]any{0, 0},
		nil,
		"merged-price",
	)
	merged := MergeLabCardSummaries(fromEval, fromMerged)
	ref := LabDetailRef("buy", 1)
	if merged[ref].PriceAnalysis != "flow-price" {
		t.Fatalf("eval row should win: %q", merged[ref].PriceAnalysis)
	}
}
