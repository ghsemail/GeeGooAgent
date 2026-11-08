package strategydev

import (
	"encoding/json"
	"strings"
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
	if detail.JEVUnified == nil || detail.JEVUnified.Direction != wantJEV.Direction || detail.JEVUnified.Confidence != wantJEV.Confidence {
		t.Fatalf("jev_unified changed: got %+v want direction/conf %+v/%v", detail.JEVUnified, wantJEV.Direction, wantJEV.Confidence)
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

func TestPrepareLabEpisodeDetailForEmit_DailyMacroKlineColumnOnly(t *testing.T) {
	dailyBar := slots.KlinePatternBarWindowEntry{
		Role:          "anchor",
		KlinePatterns: []slots.KlinePatternHit{{LabelCN: "射击之星", Direction: -1}},
	}
	daily := slots.KlinePatternMTFEntry{
		Frequency: "daily",
		Label:     "日线",
		ThreeBar: &slots.KlinePatternThreeBarWindow{
			Bars: []slots.KlinePatternBarWindowEntry{dailyBar},
		},
	}
	mtf := &slots.KlinePatternMTFContext{
		SignalFrequency: "60m",
		WeightSchema:    "mtf_v1",
		PrimaryBias:     &slots.KlinePatternBias{Bias: "看空", Score: -0.89},
		Timeframes: []slots.KlinePatternMTFEntry{
			{Frequency: "60m", Label: "1小时"},
			daily,
			{Frequency: "weekly", Label: "周线"},
		},
	}
	detail := LabEpisodeDetail{
		KlineJEVProfile: LabKlineJEVProfileDailyMacro,
		KlinePatternMTF: mtf,
		KlinePattern: &LabKlinePatternDetail{
			Summary: "多周期加权看空（-0.89）：1小时 贡献偏空",
			MTF:     mtf,
		},
	}
	PrepareLabEpisodeDetailForEmit(&detail)
	if detail.KlinePattern == nil || detail.KlinePattern.MTF == nil {
		t.Fatal("missing kline_pattern.mtf")
	}
	if len(detail.KlinePattern.MTF.Timeframes) != 1 {
		t.Fatalf("want 1 daily column, got %d: %+v", len(detail.KlinePattern.MTF.Timeframes), detail.KlinePattern.MTF.Timeframes)
	}
	if detail.KlinePattern.MTF.SignalFrequency != "daily" {
		t.Fatalf("signal_frequency=%q", detail.KlinePattern.MTF.SignalFrequency)
	}
	if strings.TrimSpace(detail.KlinePattern.MTF.WeightSchema) != "" {
		t.Fatalf("weight_schema should be empty: %q", detail.KlinePattern.MTF.WeightSchema)
	}
	if strings.Contains(detail.KlinePattern.Summary, "多周期") {
		t.Fatalf("summary should be daily hits only: %q", detail.KlinePattern.Summary)
	}
	if !strings.Contains(detail.KlinePattern.Summary, "射击之星") {
		t.Fatalf("summary=%q", detail.KlinePattern.Summary)
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
