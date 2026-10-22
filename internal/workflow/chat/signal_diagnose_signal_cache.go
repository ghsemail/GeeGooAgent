package chat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// ensureDiagnoseSignalResolved resolves catalog/panel signal once per flow (read_strategy).
func (r *Runner) ensureDiagnoseSignalResolved(
	ctx context.Context,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
	phaseTimingKey string,
) error {
	if flow == nil || flow.DiagnoseSignalCached {
		return nil
	}
	start := time.Now()
	sig, src, err := r.resolveDiagnoseSignalUncached(ctx, flow, toolCtx, recordTool)
	if err != nil {
		return err
	}
	if phaseTimingKey != "" {
		recordPhaseTiming(flow, phaseTimingKey, time.Since(start))
	}
	flow.DiagnoseSignalCached = true
	flow.DiagnoseSignalSource = src
	flow.ResolvedBuySignal = sig.Buy
	flow.ResolvedSellSignal = sig.Sell
	flow.ResolvedSignalFrequency = sig.Frequency
	flow.ResolvedStrategyLabel = sig.StrategyLabel
	flow.ProbeSignalFingerprint = fingerprintProbeSignalPlan(flow)
	return nil
}

func (r *Runner) diagnoseSignalFromFlow(
	ctx context.Context,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
) (slots.ResolvedSignal, error) {
	if flow != nil && flow.DiagnoseSignalCached {
		return slots.ResolvedSignal{
			Buy:           flow.ResolvedBuySignal,
			Sell:          flow.ResolvedSellSignal,
			Frequency:     flow.ResolvedSignalFrequency,
			StrategyLabel: flow.ResolvedStrategyLabel,
		}, nil
	}
	sig, _, err := r.resolveDiagnoseSignalUncached(ctx, flow, toolCtx, recordTool)
	return sig, err
}

func (r *Runner) resolveDiagnoseSignalUncached(
	ctx context.Context,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
) (slots.ResolvedSignal, string, error) {
	if flowHasProbePanelOverrides(flow) {
		return resolvedSignalFromProbeOverrides(flow), "panel", nil
	}
	kind := "combination"
	switch flow.CatalogType {
	case catalogTypeIndex, catalogTypeDefinition:
		kind = "indicator"
	}
	plan := slots.SignalPlan{
		SignalQuery: coalesceStrings(flow.CatalogLabel, flow.StrategyQuery),
		SignalKind:  kind,
	}
	sig, err := slots.ResolveSignal(ctx, toolCtx, r.runToolCall(recordTool), plan)
	if err != nil {
		return slots.ResolvedSignal{}, "", err
	}
	return sig, "catalog", nil
}

func fingerprintProbeSignalPlan(flow *Flow) string {
	if flow == nil {
		return ""
	}
	payload := map[string]any{
		"buy":  flow.ProbeBuyOverride,
		"sell": flow.ProbeSellOverride,
		"buy_merge":  flow.ProbeBuyMerge,
		"sell_merge": flow.ProbeSellMerge,
		"frequency":  coalesceStrings(flow.ProbeFrequencyOverride, flow.ResolvedSignalFrequency),
		"catalog":    flow.CatalogLabel,
		"catalog_type": flow.CatalogType,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:8])
}

func probeSignalPlanForCard(flow *Flow) map[string]any {
	if flow == nil {
		return nil
	}
	out := map[string]any{
		"fingerprint": flow.ProbeSignalFingerprint,
		"source":      flow.DiagnoseSignalSource,
		"frequency":   coalesceStrings(flow.ProbeFrequencyOverride, flow.ResolvedSignalFrequency),
	}
	if len(flow.ProbeBuyOverride) > 0 {
		out["buy_signal"] = flow.ProbeBuyOverride
	} else if len(flow.ResolvedBuySignal) > 0 {
		out["buy_signal"] = flow.ResolvedBuySignal
	}
	if len(flow.ProbeSellOverride) > 0 {
		out["sell_signal"] = flow.ProbeSellOverride
	} else if len(flow.ResolvedSellSignal) > 0 {
		out["sell_signal"] = flow.ResolvedSellSignal
	}
	if s := strings.TrimSpace(flow.ProbeBuyMerge); s != "" {
		out["buy_merge"] = s
	}
	if s := strings.TrimSpace(flow.ProbeSellMerge); s != "" {
		out["sell_merge"] = s
	}
	return out
}
