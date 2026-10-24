package strategydev

import (
	"encoding/json"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestPrepareLabEpisodeDetailForEmit_AnchorOnlyChartMarkers(t *testing.T) {
	detail := LabEpisodeDetail{
		KlinePatterns: []slots.KlinePatternHit{{Name: "AnchorHit", Direction: 1}},
		KlinePatternThreeBar: &slots.KlinePatternThreeBarWindow{
			Bars: []slots.KlinePatternBarWindowEntry{
				{Role: "prev2", KlinePatterns: []slots.KlinePatternHit{{Name: "Old"}}},
				{Role: "prev", KlinePatterns: []slots.KlinePatternHit{{Name: "Old2"}}},
				{Role: "anchor", KlinePatterns: []slots.KlinePatternHit{{Name: "AnchorHit"}}},
			},
		},
		KlinePattern: &LabKlinePatternDetail{
			ThreeBar: &slots.KlinePatternThreeBarWindow{
				Bars: []slots.KlinePatternBarWindowEntry{
					{Role: "prev", KlinePatterns: []slots.KlinePatternHit{{Name: "NestedPrev"}}},
					{Role: "anchor", KlinePatterns: []slots.KlinePatternHit{{Name: "NestedAnchor"}}},
				},
			},
		},
		JEVUnified: &LabJEVUnified{Direction: LabJEVUnifiedBull, Confidence: 0.8},
	}
	ApplyLabJEVUnified(&detail) // no-op overwrite; ensures contract path exists
	wantJEV := *detail.JEVUnified

	PrepareLabEpisodeDetailForEmit(&detail)

	if len(detail.KlinePatternThreeBar.Bars[0].KlinePatterns) != 0 || len(detail.KlinePatternThreeBar.Bars[1].KlinePatterns) != 0 {
		t.Fatalf("top-level three_bar prev markers=%+v", detail.KlinePatternThreeBar.Bars)
	}
	if len(detail.KlinePatternThreeBar.Bars[2].KlinePatterns) != 1 {
		t.Fatalf("anchor markers=%+v", detail.KlinePatternThreeBar.Bars[2].KlinePatterns)
	}
	if len(detail.KlinePattern.ThreeBar.Bars[0].KlinePatterns) != 0 {
		t.Fatalf("nested prev markers=%+v", detail.KlinePattern.ThreeBar.Bars)
	}
	if len(detail.KlinePatterns) != 1 {
		t.Fatalf("episode-level kline_patterns for JEV must stay: %+v", detail.KlinePatterns)
	}
	if detail.JEVUnified == nil || *detail.JEVUnified != wantJEV {
		t.Fatalf("jev_unified changed: got %+v want %+v", detail.JEVUnified, wantJEV)
	}

	raw, err := json.Marshal(detail)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	tb, _ := m["kline_pattern_three_bar"].(map[string]any)
	bars, _ := tb["bars"].([]any)
	prev2, _ := bars[0].(map[string]any)
	if _, ok := prev2["kline_patterns"]; ok {
		t.Fatalf("chart JSON must omit prev2 kline_patterns: %v", prev2)
	}
}

func TestEpisodeTriggerRecordsForChartEmit(t *testing.T) {
	list := []EpisodeTriggerRecord{{
		KlinePatternThreeBar: &slots.KlinePatternThreeBarWindow{
			Bars: []slots.KlinePatternBarWindowEntry{
				{Role: "prev", KlinePatterns: []slots.KlinePatternHit{{Name: "P"}}},
				{Role: "anchor", KlinePatterns: []slots.KlinePatternHit{{Name: "A"}}},
			},
		},
	}}
	out := EpisodeTriggerRecordsForChartEmit(list)
	if len(out[0].KlinePatternThreeBar.Bars[0].KlinePatterns) != 0 {
		t.Fatalf("trigger emit prev=%+v", out[0].KlinePatternThreeBar.Bars[0])
	}
	if len(list[0].KlinePatternThreeBar.Bars[0].KlinePatterns) != 1 {
		t.Fatal("source trigger must not be mutated")
	}
}
