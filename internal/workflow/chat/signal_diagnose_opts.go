package chat

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

const (
	MetaKeyStickyPanelOps        = "sticky_panel_ops"
	metaKeyLegacyStickyPanelOps  = "signal_diagnose_sticky_opts"
)

// ApplyInboundPanelOpsToFlow merges sticky + inbound panel ops onto a new signal_diagnose flow.
// Sticky survives across turns so toggling lab_mode alone cannot drop buy/sell overrides.
func ApplyInboundPanelOpsToFlow(session *runtime.Session, flow *Flow) {
	if session == nil || flow == nil {
		return
	}
	merged := MergeSignalDiagnoseOpts(session.StickyPanelOps, session.InboundPanelOps)
	if merged == nil {
		return
	}
	session.StickyPanelOps = runtime.CloneSignalDiagnoseOpts(merged)
	applySignalDiagnoseOptsToFlow(merged, flow)
	session.InboundPanelOps = nil
}

func signalDiagnoseEpisodeEvalOpts(flow *Flow) slots.EvaluateSignalEpisodesOptions {
	if flow != nil && flow.TradeDirection.LongOnlyEpisodeEval() {
		return slots.EvaluateSignalEpisodesOptions{LongOnly: true}
	}
	return slots.EvaluateSignalEpisodesOptions{}
}

// effectiveSignalDiagnoseBuyScoreDirection is the soft price-analysis buy intercept mode
// when unset on a diagnose flow (production default: reverse).
func effectiveSignalDiagnoseBuyScoreDirection(d strategydev.BuyScoreDirection) strategydev.BuyScoreDirection {
	if strings.TrimSpace(string(d)) == "" {
		return strategydev.BuyScoreDirectionReverse
	}
	return d
}

// ParseSignalDiagnoseOptsFromSkillOptions reads skill_options.signal_diagnose (or legacy workflow_options bag).
func ParseSignalDiagnoseOptsFromSkillOptions(raw map[string]any) *runtime.SignalDiagnoseOpts {
	return parseSignalDiagnoseOptsFromOptionsBag(raw)
}

// ParseSignalDiagnoseOptsFromWorkflowOptions reads legacy workflow_options.signal_diagnose from chat request.
func ParseSignalDiagnoseOptsFromWorkflowOptions(raw map[string]any) *runtime.SignalDiagnoseOpts {
	return parseSignalDiagnoseOptsFromOptionsBag(raw)
}

func parseSignalDiagnoseOptsFromOptionsBag(raw map[string]any) *runtime.SignalDiagnoseOpts {
	if len(raw) == 0 {
		return nil
	}
	block, _ := raw["signal_diagnose"].(map[string]any)
	if block == nil {
		return nil
	}
	out := &runtime.SignalDiagnoseOpts{}
	// Legacy clients may still send use_key_level_episode_stop; ignored (feature removed).
	if v, ok := block["use_price_analysis_episode_filter"].(bool); ok {
		out.UsePriceAnalysisEpisodeFilter = &v
	}
	if v, ok := block["price_analysis_score_threshold"].(float64); ok {
		out.PriceAnalysisScoreThreshold = &v
	}
	if s, ok := block["price_analysis_buy_score_direction"].(string); ok && strings.TrimSpace(s) != "" {
		out.PriceAnalysisBuyScoreDirection = strings.TrimSpace(s)
	}
	if s, ok := block["price_analysis_gate_score_mode"].(string); ok && strings.TrimSpace(s) != "" {
		out.PriceAnalysisGateScoreMode = strings.TrimSpace(s)
	}
	if raw, ok := block["buy_signal"].([]any); ok && len(raw) > 0 {
		out.BuySignal = raw
	}
	if raw, ok := block["sell_signal"].([]any); ok && len(raw) > 0 {
		out.SellSignal = raw
	}
	if s, ok := block["sell_merge"].(string); ok && strings.TrimSpace(s) != "" {
		out.SellMerge = strings.TrimSpace(s)
	}
	if s, ok := block["buy_merge"].(string); ok && strings.TrimSpace(s) != "" {
		out.BuyMerge = strings.TrimSpace(s)
	}
	if s, ok := block["frequency"].(string); ok {
		out.Frequency = strings.TrimSpace(s)
	}
	if v, ok := block["months_back"].(float64); ok && v > 0 {
		out.MonthsBack = int(v)
	} else if v, ok := block["months_back"].(int); ok && v > 0 {
		out.MonthsBack = v
	}
	if s, ok := block["panel_strategy_label"].(string); ok {
		out.PanelStrategyLabel = strings.TrimSpace(s)
	}
	if _, ok := block["trade_direction"]; ok {
		s, _ := block["trade_direction"].(string)
		td := runtime.ParseTradeDirection(s)
		out.TradeDirection = &td
	}
	if v, ok := block["lab_mode"].(bool); ok {
		out.LabMode = &v
	}
	return out
}
