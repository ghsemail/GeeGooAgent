package strategydev

import (
	"encoding/json"
	"testing"

	kp "github.com/ghsemail/GeeGooAgent/internal/lab/klinepattern"
)

func TestKlinePatternDisplayDisabled_OmitsTriggerFields(t *testing.T) {
	SetKlinePatternDisplayEnabledForTest(t, false)
	series := []any{[]any{map[string]any{"name": "Hammer", "direction": 1}}}
	if kp.FilteredKlinePatternsAtIndex(series, 0) == nil {
		t.Fatal("archive parser should still work; prod gate is in bridge")
	}
	rec := &EpisodeTriggerRecord{Index: 0}
	EnrichEpisodeTriggerRecordPatterns(rec, series)
	if len(rec.KlinePatterns) != 0 || rec.KlinePatternSummary != "" {
		t.Fatalf("rec=%+v", rec)
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "{}" && string(raw) != "{\"index\":0}" {
		var m map[string]any
		_ = json.Unmarshal(raw, &m)
		if _, ok := m["kline_patterns"]; ok {
			t.Fatalf("unexpected kline_patterns in %s", string(raw))
		}
	}
}

func TestEnrichTriggerKlineDisplay_WeightedTopLevelWhenMTFOnlyHits(t *testing.T) {
	SetKlinePatternDisplayEnabledForTest(t, true)
	bars := []any{
		map[string]any{"time": "2024-01-01", "open": 1, "high": 2, "low": 0.5, "close": 1.5},
		map[string]any{"time": "2024-01-02", "open": 1.5, "high": 2.5, "low": 1, "close": 2},
	}
	hits := []any{nil, nil}
	higherBars := []any{
		map[string]any{"time": "2024-01-01", "close": 1},
		map[string]any{"time": "2024-01-02", "close": 2},
	}
	higherHits := []any{
		nil,
		[]any{map[string]any{"name": "Engulfing", "direction": -1}},
	}
	in := NewKlinePatternDisplayInput("5m", bars, nil, &KlinePatternMTFBundle{
		SignalFreq: "5m",
		ByFreq: map[string]KlinePatternFreqSeries{
			"15m": {Freq: "15m", Bars: higherBars, PatternHits: higherHits, FetchOK: true},
			"30m": {Freq: "30m", FetchOK: false},
			"60m": {Freq: "60m", FetchOK: false},
		},
	})
	in.PatternHits = hits
	rec := &EpisodeTriggerRecord{Index: 1}
	enrichTriggerKlineDisplay(rec, in)
	if len(rec.KlinePatterns) != 0 {
		t.Fatalf("expected empty top-level patterns, got %+v", rec.KlinePatterns)
	}
	if rec.KlinePatternSummary == "" {
		t.Fatal("expected weighted summary when MTF has hits")
	}
	if rec.KlinePatternBias == nil || rec.KlinePatternBias.Score >= 0 {
		t.Fatalf("expected bearish weighted bias, got %+v", rec.KlinePatternBias)
	}
	if rec.KlinePatternMTF == nil || rec.KlinePatternMTF.WeightSchema != kp.KlinePatternMTFWeightSchema {
		t.Fatalf("mtf=%+v", rec.KlinePatternMTF)
	}
}

func TestClampProbeSeriesToMonthsBack_SlicesPatternHits(t *testing.T) {
	data := map[string]any{
		"bars": []any{
			map[string]any{"time": "2023-10-01", "close": 1},
			map[string]any{"time": "2024-01-01", "close": 2},
			map[string]any{"time": "2024-02-01", "close": 3},
		},
		"buy_merged":   []any{0, 1, 0},
		"sell_merged":  []any{0, 0, 0},
		"pattern_hits": []any{"a", "b", "c"},
		"as_of":        "2024-02-01",
	}
	ClampProbeSeriesToMonthsBack(data, 3)
	hits, _ := data["pattern_hits"].([]any)
	bars, _ := data["bars"].([]any)
	if len(hits) != len(bars) {
		t.Fatalf("pattern_hits=%d bars=%d", len(hits), len(bars))
	}
}
