package chat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

const (
	resolveCacheScopeMiss    int64 = 0
	resolveCacheScopeSession int64 = 1
	resolveCacheScopeGlobal  int64 = 2
)

// ensureDiagnoseSignalResolved resolves catalog/panel signal once per flow (read_strategy).
func (r *Runner) ensureDiagnoseSignalResolved(
	ctx context.Context,
	session *runtime.Session,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
	phaseTimingKey string,
) error {
	if flow == nil || flow.DiagnoseSignalCached {
		return nil
	}
	cacheKey := signalResolveCacheKey(flow)
	flow.SignalResolveCacheKey = cacheKey
	if ent, ok := loadSignalResolveFromSession(session, cacheKey); ok {
		applySignalResolveCacheEntry(flow, ent)
		recordResolveCacheMetrics(flow, phaseTimingKey, resolveCacheScopeSession)
		return nil
	}
	if ent, ok := loadSignalResolveGlobal(cacheKey); ok {
		applySignalResolveCacheEntry(flow, ent)
		persistResolvedSignalCache(session, cacheKey, slots.ResolvedSignal{
			Buy: ent.Buy, Sell: ent.Sell, Frequency: ent.Frequency, StrategyLabel: ent.StrategyLabel,
		}, ent.Source)
		recordResolveCacheMetrics(flow, phaseTimingKey, resolveCacheScopeGlobal)
		return nil
	}
	start := time.Now()
	sig, src, resolvePath, err := r.resolveDiagnoseSignalUncached(ctx, flow, toolCtx, recordTool)
	if err != nil {
		return err
	}
	elapsed := time.Since(start)
	if phaseTimingKey != "" {
		recordPhaseTiming(flow, phaseTimingKey, elapsed)
		recordPhaseGauge(flow, phaseTimingKey+":cache_hit", 0)
		recordPhaseGauge(flow, phaseTimingKey+":cache_scope", resolveCacheScopeMiss)
		if resolvePath != "" {
			recordPhaseTiming(flow, phaseTimingKey+":"+resolvePath, elapsed)
		}
	}
	flow.DiagnoseSignalCached = true
	flow.DiagnoseSignalSource = src
	flow.ResolvedBuySignal = sig.Buy
	flow.ResolvedSellSignal = sig.Sell
	flow.ResolvedSignalFrequency = sig.Frequency
	flow.ResolvedStrategyLabel = sig.StrategyLabel
	flow.ProbeSignalFingerprint = fingerprintProbeSignalPlan(flow)
	persistResolvedSignalCache(session, cacheKey, sig, src)
	return nil
}

func recordResolveCacheMetrics(flow *Flow, phaseTimingKey string, scope int64) {
	if phaseTimingKey == "" {
		return
	}
	recordPhaseGauge(flow, phaseTimingKey+":cache_hit", 1)
	recordPhaseGauge(flow, phaseTimingKey+":cache_scope", scope)
	recordPhaseTiming(flow, phaseTimingKey+":cached", time.Millisecond)
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
	sig, _, _, err := r.resolveDiagnoseSignalUncached(ctx, flow, toolCtx, recordTool)
	return sig, err
}

func (r *Runner) resolveDiagnoseSignalUncached(
	ctx context.Context,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
) (slots.ResolvedSignal, string, string, error) {
	if flowHasProbePanelOverrides(flow) {
		return resolvedSignalFromProbeOverrides(flow), "panel", "resolve_panel", nil
	}
	if len(flow.CatalogRaw) > 0 {
		sig, err := slots.ResolvedSignalFromCatalog(flow.CatalogType, flow.CatalogRaw)
		if err == nil && len(sig.Buy) > 0 {
			return sig, "catalog_raw", "resolve_catalog_raw", nil
		}
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
	runTool := cachedCatalogToolRunner(r.runToolCall(recordTool))
	sig, err := slots.ResolveSignal(ctx, toolCtx, runTool, plan)
	if err != nil {
		return slots.ResolvedSignal{}, "", "resolve_mcp", err
	}
	return sig, "catalog", "resolve_mcp", nil
}

func fingerprintProbeSignalPlan(flow *Flow) string {
	if flow == nil {
		return ""
	}
	payload := map[string]any{
		"buy":            flow.ProbeBuyOverride,
		"sell":           flow.ProbeSellOverride,
		"buy_merge":      flow.ProbeBuyMerge,
		"sell_merge":     flow.ProbeSellMerge,
		"frequency":      coalesceStrings(flow.ProbeFrequencyOverride, flow.ResolvedSignalFrequency),
		"catalog":        flow.CatalogLabel,
		"catalog_type":   flow.CatalogType,
		"resolved_buy":   flow.ResolvedBuySignal,
		"resolved_sell":  flow.ResolvedSellSignal,
		"strategy_query": flow.StrategyQuery,
		"months_back":    flow.MonthsBack,
	}
	if len(flow.CatalogRaw) > 0 {
		payload["catalog_signal_id"] = catalogField(flow.CatalogRaw, "signal_id", "id")
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
	if k := strings.TrimSpace(flow.SignalResolveCacheKey); k != "" {
		out["resolve_cache_key"] = k
	}
	if len(flow.CatalogRaw) > 0 {
		if id := catalogField(flow.CatalogRaw, "signal_id", "id"); id != "" {
			out["catalog_signal_id"] = id
		}
		if name := catalogField(flow.CatalogRaw, "name"); name != "" {
			out["catalog_name"] = name
		}
	}
	if flowHasProbePanelOverrides(flow) {
		out["rule_origin"] = "panel"
	} else {
		switch flow.DiagnoseSignalSource {
		case "catalog_raw":
			out["rule_origin"] = "catalog"
		case "panel":
			out["rule_origin"] = "panel"
		default:
			out["rule_origin"] = "catalog_mcp"
		}
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
