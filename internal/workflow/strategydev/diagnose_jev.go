package strategydev

import (
	"context"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// DiagnoseJEVUnifiedPlan selects participating JEV lanes from converted diagnose facts (no lab-api client gates).
func DiagnoseJEVUnifiedPlan(out *LabEpisodeDetail) labJEVUnifiedPlan {
	plan := labJEVUnifiedPlan{}
	if labTAInformative(out.TechnicalAnalysis) {
		plan.attemptTA = true
	}
	if labIndexTrendInformative(out.IndexTrend) {
		plan.attemptIndexTrend = true
	}
	if out.PriceAnalysis != nil && labPAInformative(out.PriceAnalysis) {
		plan.attemptPA = true
	}
	if kline := labJEVKlineFromDetail(out); !labJEVKlineInputEmpty(kline) {
		plan.attemptKline = true
	}
	return plan
}

// ApplyDiagnoseJEVUnified runs total JEV for diagnose skills (four-lane facts already on detail).
func ApplyDiagnoseJEVUnified(ctx context.Context, out *LabEpisodeDetail, jev *JevClient, signalFreq, klineProfile string) {
	if out == nil {
		return
	}
	plan := DiagnoseJEVUnifiedPlan(out)
	if labJEVUnifiedBlocked(out, plan) {
		reason := labJEVUnifiedBlockReason(out, plan)
		fail := emptyLabJEVUnified(config.DefaultDecisionModel)
		if jev != nil && strings.TrimSpace(jev.cfg.Model) != "" {
			fail.Model = strings.TrimSpace(jev.cfg.Model)
		}
		fail.Error = reason
		out.JEVUnified = &fail
		return
	}
	if klineProfile != "" {
		out.KlineJEVProfile = klineProfile
	}
	rec := ResolveLabJEVUnified(ctx, jev, out, plan, "", signalFreq, false)
	out.JEVUnified = &rec
}

// LabPAInformative reports whether converted diagnose price analysis can run JEV.
func LabPAInformative(pa *PriceAnalysis) bool {
	return labPAInformative(pa)
}

// LabJEVKlineInputFromDetail exposes kline JEV facts for diagnose skills.
func LabJEVKlineInputFromDetail(out *LabEpisodeDetail) *LabJEVKlineInput {
	return labJEVKlineFromDetail(out)
}

// LabJEVKlineInputIsEmpty mirrors labJEVKlineInputEmpty for external packages.
func LabJEVKlineInputIsEmpty(k *LabJEVKlineInput) bool {
	return labJEVKlineInputEmpty(k)
}

// MacroSessionSpecForExport exposes exchange session metadata for macro monitor close.
func MacroSessionSpecForExport(code string) (macroSessionSpec, bool) {
	return macroSessionSpecForCode(code)
}

// MacroSessionCloseInstantExport exposes session close instant for a calendar day.
func MacroSessionCloseInstantExport(day time.Time, spec macroSessionSpec) time.Time {
	return sessionCloseInstant(day, spec)
}