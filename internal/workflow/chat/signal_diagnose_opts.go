package chat

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

const (
	KeyBreakModeSupportLow = "support_low"
	KeyBreakModeResistHigh = "resist_high"

	KeyBreakRefSupportLow    = "support_low"
	KeyBreakRefSupportHigh   = "support_high"
	KeyBreakRefSupportCenter = "support_center"
	KeyBreakRefResistLow     = "resist_low"
	KeyBreakRefResistHigh    = "resist_high"
	KeyBreakRefResistCenter  = "resist_center"
)

// ApplyPendingSignalDiagnoseOpts copies session pending opts onto a new flow and clears pending.
func ApplyPendingSignalDiagnoseOpts(session *runtime.Session, flow *Flow) {
	if session == nil || flow == nil || session.PendingSignalDiagnoseOpts == nil {
		return
	}
	o := session.PendingSignalDiagnoseOpts
	flow.UseKeyLevelEpisodeStop = o.UseKeyLevelEpisodeStop
	flow.KeyBreakMode = normalizeKeyBreakMode(o.KeyBreakMode)
	flow.KeyBreakBuyRef = normalizeKeyBreakRef(o.KeyBreakBuyRef, defaultBuyBreakRef(o.KeyBreakMode))
	flow.KeyBreakSellRef = normalizeKeyBreakRef(o.KeyBreakSellRef, defaultSellBreakRef(o.KeyBreakMode))
	session.PendingSignalDiagnoseOpts = nil
}

// ParseSignalDiagnoseOptsFromWorkflowOptions reads workflow_options.signal_diagnose from chat request.
func ParseSignalDiagnoseOptsFromWorkflowOptions(raw map[string]any) *runtime.SignalDiagnoseOpts {
	if len(raw) == 0 {
		return nil
	}
	block, _ := raw["signal_diagnose"].(map[string]any)
	if block == nil {
		return nil
	}
	out := &runtime.SignalDiagnoseOpts{}
	if v, ok := block["use_key_level_episode_stop"].(bool); ok {
		out.UseKeyLevelEpisodeStop = v
	}
	if s, ok := block["key_break_mode"].(string); ok {
		out.KeyBreakMode = normalizeKeyBreakMode(s)
	}
	if s, ok := block["key_break_buy_ref"].(string); ok {
		out.KeyBreakBuyRef = normalizeKeyBreakRef(s, "")
	}
	if s, ok := block["key_break_sell_ref"].(string); ok {
		out.KeyBreakSellRef = normalizeKeyBreakRef(s, "")
	}
	return out
}

// DefaultSignalDiagnoseOptsForEval enables key-level episode stop for dashboard eval cases.
func DefaultSignalDiagnoseOptsForEval() *runtime.SignalDiagnoseOpts {
	return &runtime.SignalDiagnoseOpts{
		UseKeyLevelEpisodeStop: true,
		KeyBreakMode:           KeyBreakModeSupportLow,
	}
}

func normalizeKeyBreakMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case KeyBreakModeResistHigh:
		return KeyBreakModeResistHigh
	default:
		return KeyBreakModeSupportLow
	}
}

func defaultBuyBreakRef(mode string) string {
	if normalizeKeyBreakMode(mode) == KeyBreakModeResistHigh {
		return KeyBreakRefResistHigh
	}
	return KeyBreakRefSupportLow
}

func defaultSellBreakRef(mode string) string {
	return KeyBreakRefResistHigh
}

func normalizeKeyBreakRef(ref, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(ref)) {
	case KeyBreakRefSupportLow, KeyBreakRefSupportHigh, KeyBreakRefSupportCenter,
		KeyBreakRefResistLow, KeyBreakRefResistHigh, KeyBreakRefResistCenter:
		return strings.ToLower(strings.TrimSpace(ref))
	default:
		if fallback != "" {
			return fallback
		}
		return KeyBreakRefSupportLow
	}
}
