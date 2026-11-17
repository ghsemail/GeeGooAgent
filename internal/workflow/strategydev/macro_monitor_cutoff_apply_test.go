package strategydev

import (
	"strings"
	"testing"
	"time"
)

func TestApplyMacroMonitorCutoff_UnifiesPADisplayAnchor(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Date(2026, 11, 16, 16, 30, 0, 0, loc)
	bars := []any{
		map[string]any{"time": "2026-11-16 00:00:00", "close": 3979.8},
	}
	cut, ok := resolveMacroMonitorCutoffFromDailyBars("000300.SH", now, bars)
	if !ok {
		t.Fatal("cutoff")
	}
	out := LabEpisodeDetail{
		KlineJEVProfile: LabKlineJEVProfileDailyMacro,
		PriceAnalysis:   &PriceAnalysis{},
		TechnicalAnalysis: &LabTechnicalAnalysis{},
	}
	applyMacroMonitorCutoffToDetail(&out, cut)
	applyDailyMacroLabHeaderTime(&out)
	ta := &LabTechnicalAnalysis{AnchorTime: cut.DisplayTime}
	out.TechnicalAnalysis = ta
	if out.MonitorQuoteCutoffDay != "2026-11-16" {
		t.Fatalf("day=%q", out.MonitorQuoteCutoffDay)
	}
	if !strings.Contains(out.PriceAnalysis.AnchorTime, "2026-11-16") {
		t.Fatalf("pa anchor=%q", out.PriceAnalysis.AnchorTime)
	}
	if !strings.Contains(out.Time, "2026-11-16") {
		t.Fatalf("header time=%q", out.Time)
	}
}