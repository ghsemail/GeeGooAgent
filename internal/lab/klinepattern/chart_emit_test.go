package klinepattern

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestThreeBarWindowForChartEmit_KeepsAnchorHitsOnly(t *testing.T) {
	win := &slots.KlinePatternThreeBarWindow{
		AnchorIndex: 2,
		Bars: []slots.KlinePatternBarWindowEntry{
			{Role: "prev2", Index: 0, KlinePatterns: []slots.KlinePatternHit{{Name: "Hammer", Direction: 1}}},
			{Role: "prev", Index: 1, KlinePatterns: []slots.KlinePatternHit{{Name: "Doji", Direction: 2}}},
			{Role: "anchor", Index: 2, KlinePatterns: []slots.KlinePatternHit{{Name: "ThreeWhiteSoldiers", Direction: 1}}},
		},
		WindowBias: &slots.KlinePatternBias{Score: 2, Bias: "看多"},
	}
	emit := ThreeBarWindowForChartEmit(win)
	if len(emit.Bars) != 3 {
		t.Fatalf("bars=%d", len(emit.Bars))
	}
	if len(emit.Bars[0].KlinePatterns) != 0 || len(emit.Bars[1].KlinePatterns) != 0 {
		t.Fatalf("prev2/prev must not carry chart markers: %+v", emit.Bars)
	}
	if len(emit.Bars[2].KlinePatterns) != 1 || emit.Bars[2].KlinePatterns[0].Name != "ThreeWhiteSoldiers" {
		t.Fatalf("anchor patterns=%+v", emit.Bars[2].KlinePatterns)
	}
	if emit.WindowBias == nil || emit.WindowBias.Score != 2 {
		t.Fatalf("window_bias must be preserved for summaries, got %+v", emit.WindowBias)
	}
	// Source window unchanged (emit is a copy).
	if len(win.Bars[0].KlinePatterns) != 1 {
		t.Fatal("source prev2 patterns should remain for internal scoring paths")
	}
}

func TestMTFContextForChartEmit_StripsNestedThreeBarMarkers(t *testing.T) {
	ctx := &slots.KlinePatternMTFContext{
		Timeframes: []slots.KlinePatternMTFEntry{{
			Label: "信号",
			ThreeBar: &slots.KlinePatternThreeBarWindow{
				Bars: []slots.KlinePatternBarWindowEntry{
					{Role: "prev", KlinePatterns: []slots.KlinePatternHit{{Name: "X"}}},
					{Role: "anchor", KlinePatterns: []slots.KlinePatternHit{{Name: "Y"}}},
				},
			},
		}},
	}
	emit := MTFContextForChartEmit(ctx)
	if len(emit.Timeframes[0].ThreeBar.Bars[0].KlinePatterns) != 0 {
		t.Fatalf("mtf prev markers=%+v", emit.Timeframes[0].ThreeBar.Bars[0].KlinePatterns)
	}
	if len(emit.Timeframes[0].ThreeBar.Bars[1].KlinePatterns) != 1 {
		t.Fatalf("mtf anchor markers=%+v", emit.Timeframes[0].ThreeBar.Bars[1].KlinePatterns)
	}
}
