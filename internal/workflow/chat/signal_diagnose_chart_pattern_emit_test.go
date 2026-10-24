package chat

import (
	"encoding/json"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestChartProbePayload_ThreeBarAnchorOnlyPatternMarkers(t *testing.T) {
	threeBar := &slots.KlinePatternThreeBarWindow{
		AnchorIndex: 1,
		Bars: []slots.KlinePatternBarWindowEntry{
			{Role: "prev", Index: 0, KlinePatterns: []slots.KlinePatternHit{{Name: "PrevHit"}}},
			{Role: "anchor", Index: 1, KlinePatterns: []slots.KlinePatternHit{{Name: "AnchorHit"}}},
		},
	}
	flow := &Flow{
		ProbeRaw: map[string]any{
			"bars": []any{
				map[string]any{"time": "t0", "close": 1},
				map[string]any{"time": "t1", "close": 2},
			},
		},
		SignalEval: slots.SignalEpisodeEval{
			BuyDetails: []slots.SignalEpisodeDetail{{
				StartIdx: 1, StartTime: "t1", StartClose: 2,
				KlinePatterns:        []slots.KlinePatternHit{{Name: "AnchorHit"}},
				KlinePatternThreeBar: threeBar,
			}},
		},
	}
	cp := chartProbePayload(flow)
	raw, err := json.Marshal(cp)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	eps, _ := m["episodes"].([]any)
	ep0, _ := eps[0].(map[string]any)
	tb, _ := ep0["kline_pattern_three_bar"].(map[string]any)
	bars, _ := tb["bars"].([]any)
	prev, _ := bars[0].(map[string]any)
	if _, ok := prev["kline_patterns"]; ok {
		t.Fatalf("chart episode prev must not expose kline_patterns: %v", prev)
	}
	anchor, _ := bars[1].(map[string]any)
	if anchor["kline_patterns"] == nil {
		t.Fatalf("anchor must keep kline_patterns: %v", anchor)
	}
	if len(threeBar.Bars[0].KlinePatterns) != 1 {
		t.Fatal("in-memory episode three_bar must remain full for non-chart paths")
	}
}
