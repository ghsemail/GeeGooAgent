package strategydev

import "testing"

func TestApplyDailyMacroLabHeaderTimeUsesPriceAnchor(t *testing.T) {
	out := &LabEpisodeDetail{
		Time:  "2026-11-04 14:00 (Asia/Shanghai)",
		Close: 3911.08,
		PriceAnalysis: &PriceAnalysis{
			AnchorTime: "2026-11-14 16:00 (Asia/Shanghai)",
			SpotPrice:  3920.5,
		},
		KlineJEVProfile: LabKlineJEVProfileDailyMacro,
	}
	applyDailyMacroLabHeaderTime(out)
	if out.Time != "2026-11-14 16:00 (Asia/Shanghai)" {
		t.Fatalf("time=%q", out.Time)
	}
	if out.Close != 3920.5 {
		t.Fatalf("close=%v", out.Close)
	}
}