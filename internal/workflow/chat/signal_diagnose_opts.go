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
	if len(o.BuySignal) > 0 {
		flow.ProbeBuyOverride = o.BuySignal
	}
	if len(o.SellSignal) > 0 {
		flow.ProbeSellOverride = o.SellSignal
	}
	if strings.TrimSpace(o.Frequency) != "" {
		flow.ProbeFrequencyOverride = strings.TrimSpace(o.Frequency)
	}
	if o.MonthsBack > 0 {
		flow.MonthsBack = o.MonthsBack
	}
	if strings.TrimSpace(o.PanelStrategyLabel) != "" {
		flow.ProbePanelLabel = strings.TrimSpace(o.PanelStrategyLabel)
	}
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
	if raw, ok := block["buy_signal"].([]any); ok && len(raw) > 0 {
		out.BuySignal = raw
	}
	if raw, ok := block["sell_signal"].([]any); ok && len(raw) > 0 {
		out.SellSignal = raw
	}
	if s, ok := block["frequency"].(string); ok {
		out.Frequency = strings.TrimSpace(s)
	}
	if v, ok := block["months_back"].(float64); ok && v > 0 {
		out.MonthsBack = int(v)
	} else 	if v, ok := block["months_back"].(int); ok && v > 0 {
		out.MonthsBack = v
	}
	if s, ok := block["panel_strategy_label"].(string); ok {
		out.PanelStrategyLabel = strings.TrimSpace(s)
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
	// 买段：low 跌破支撑上沿（daily as-of）→ Strict 提前结束。
	return KeyBreakRefSupportHigh
}

func defaultSellBreakRef(mode string) string {
	// 卖段：high 突破阻力下沿（daily as-of）→ Strict 提前结束。
	return KeyBreakRefResistLow
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
