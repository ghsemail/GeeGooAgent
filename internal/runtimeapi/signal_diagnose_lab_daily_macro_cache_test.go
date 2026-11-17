package runtimeapi_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func TestDailyMacroCacheAcceptable_RejectsStaleBeforeHTTPPath(t *testing.T) {
	stale := strategydev.LabEpisodeDetail{
		KlineJEVProfile: strategydev.LabKlineJEVProfileDailyMacro,
		MonitorQuoteCutoffDisplay: "2026-11-16 15:00 (Asia/Shanghai)",
		LabDetailCacheContract:    0,
	}
	if strategydev.LabDetailCacheAcceptable(stale, strategydev.LabKlineJEVProfileDailyMacro) {
		t.Fatal("pre-contract daily_macro cache must not be served without rebuild")
	}
}