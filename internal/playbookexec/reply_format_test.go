package playbookexec

import "testing"

func TestFormatSignalProbeReplyFallback(t *testing.T) {
	reply := formatSignalProbeReplyFallback(signalProbeReplyInput{
		Name:          "腾讯控股",
		Code:          "00700.HK",
		StrategyLabel: "SAR抛物线",
		Frequency:     "60m",
		MonthsBack:    3,
		Probe: probeSummary{
			BuyHits:     5,
			SellHits:    4,
			RangeStart:  "2026-05-01",
			RangeEnd:    "2026-08-01",
			RecentSignals: []map[string]any{
				{"time": "2026-07-01", "direction": "买入", "close": 380},
			},
		},
	})
	if !stringsContains(reply, "信号测试") {
		t.Fatalf("missing title: %s", reply)
	}
	if !stringsContains(reply, "60分钟K线") || !stringsContains(reply, "近3个月") {
		t.Fatalf("missing human settings: %s", reply)
	}
	if stringsContains(reply, "months_back") || stringsContains(reply, "frequency") {
		t.Fatalf("should not expose internal params: %s", reply)
	}
}

func TestHumanTradeConfigSummary(t *testing.T) {
	summary := humanTradeConfigSummary(defaultSmartTradeTradeConfig(), 100)
	if !stringsContains(summary, "止盈") || !stringsContains(summary, "止损") {
		t.Fatalf("summary=%q", summary)
	}
}
