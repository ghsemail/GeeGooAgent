package chat

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

const (
	KeyBreakModeSupportLow = "support_low"
	KeyBreakModeResistHigh = "resist_high"
)

// ApplyPendingSignalDiagnoseOpts copies session pending opts onto a new flow and clears pending.
func ApplyPendingSignalDiagnoseOpts(session *runtime.Session, flow *Flow) {
	if session == nil || flow == nil || session.PendingSignalDiagnoseOpts == nil {
		return
	}
	o := session.PendingSignalDiagnoseOpts
	flow.UseKeyLevelEpisodeStop = o.UseKeyLevelEpisodeStop
	flow.KeyBreakMode = normalizeKeyBreakMode(o.KeyBreakMode)
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
