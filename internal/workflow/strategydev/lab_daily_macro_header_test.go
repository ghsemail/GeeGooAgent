package strategydev

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestApplyDailyMacroLabHeaderTimeUsesKlineAnchorNotPriceAnchor(t *testing.T) {
	out := &LabEpisodeDetail{
		Time:            "2026-11-04 14:00 (Asia/Shanghai)",
		Close:           3911.08,
		KlineJEVProfile: LabKlineJEVProfileDailyMacro,
		PriceAnalysis: &PriceAnalysis{
			AnchorTime: "2026-11-10 16:00 (Asia/Shanghai)",
			SpotPrice:  3688.49,
		},
		KlinePatternThreeBar: &slots.KlinePatternThreeBarWindow{
			Bars: []slots.KlinePatternBarWindowEntry{
				{Role: "anchor", Time: "2026-11-14 00:00:00", Close: 3602.57},
			},
		},
	}
	applyDailyMacroLabHeaderTime(out)
	if out.Time != "2026-11-14 00:00:00" {
		t.Fatalf("time=%q want kline anchor", out.Time)
	}
	if out.Close != 3602.57 {
		t.Fatalf("close=%v", out.Close)
	}
}