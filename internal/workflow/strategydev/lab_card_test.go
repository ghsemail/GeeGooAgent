package strategydev

import (
	"encoding/json"
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
	out := BuildLabCardSummariesFromMerged(bars, buy, sell, nil, "", false)
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

func TestBuildLabCardSummariesFromEval_RefUsesStartIdxNotEpisodeIndex(t *testing.T) {
	eval := slots.SignalEpisodeEval{
		SellDetails: make([]slots.SignalEpisodeDetail, 8),
	}
	for i := range eval.SellDetails {
		eval.SellDetails[i].StartIdx = 100 + i*10
	}
	eval.SellDetails[7].StartIdx = 190
	out := BuildLabCardSummariesFromEval(eval, nil, "", false)
	ref := LabDetailRef("sell", 190)
	sum, ok := out[ref]
	if !ok {
		t.Fatalf("missing %q keys=%v", ref, out)
	}
	if sum.EpisodeIndex == nil || *sum.EpisodeIndex != 7 {
		t.Fatalf("episode_index=%v want 7", sum.EpisodeIndex)
	}
	if sum.StartIdx != 190 {
		t.Fatalf("start_idx=%d", sum.StartIdx)
	}
	if _, ok := out[LabDetailRef("sell", 7)]; ok {
		t.Fatal("sell:7 must not exist when episode 7 starts at bar 190")
	}
}

func TestBuildLabCardSummariesFromEval_EpisodeIndexZeroInJSON(t *testing.T) {
	eval := slots.SignalEpisodeEval{
		SellDetails: []slots.SignalEpisodeDetail{{StartIdx: 77}},
	}
	out := BuildLabCardSummariesFromEval(eval, nil, "", false)
	sum := out[LabDetailRef("sell", 77)]
	raw, err := json.Marshal(sum)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	v, ok := m["episode_index"]
	if !ok {
		t.Fatalf("first sell episode must include episode_index=0, json=%s", string(raw))
	}
	if f, ok := v.(float64); !ok || int(f) != 0 {
		t.Fatalf("episode_index=%v want 0", v)
	}
}

func TestBuildLabCardSummariesFromMergedLongOnlyOmitsSell(t *testing.T) {
	bars := []any{map[string]any{"close": 1.0}, map[string]any{"close": 2.0}}
	buy := []any{0, 1}
	sell := []any{0, -1}
	out := BuildLabCardSummariesFromMerged(bars, buy, sell, nil, "", true)
	if _, ok := out[LabDetailRef("sell", 1)]; ok {
		t.Fatal("long_only must omit sell trigger refs")
	}
	if _, ok := out[LabDetailRef("buy", 1)]; !ok {
		t.Fatal("expected buy ref")
	}
}

func TestMergeLabCardSummariesEvalWins(t *testing.T) {
	eval := slots.SignalEpisodeEval{
		BuyDetails: []slots.SignalEpisodeDetail{{StartIdx: 1}},
	}
	fromEval := BuildLabCardSummariesFromEval(eval, nil, "flow-price", false)
	fromMerged := BuildLabCardSummariesFromMerged(
		[]any{map[string]any{}, map[string]any{}},
		[]any{0, 1},
		[]any{0, 0},
		nil,
		"merged-price",
		false,
	)
	merged := MergeLabCardSummaries(fromEval, fromMerged)
	ref := LabDetailRef("buy", 1)
	if merged[ref].PriceAnalysis != "flow-price" {
		t.Fatalf("eval row should win: %q", merged[ref].PriceAnalysis)
	}
}
