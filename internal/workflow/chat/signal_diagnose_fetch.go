package chat

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

// applySignalDiagnoseSeriesData normalizes probe/diagnose tool payload onto the flow.
// Probe series (hits, ProbeRaw) always come from the probe tool path; diagnose merge must not overwrite them.
func applySignalDiagnoseSeriesData(
	flow *Flow,
	res tools.Result,
	months int,
	buySignal, sellSignal []any,
	fromDiagnoseTool bool,
) strategydev.IndicatorWarmupMeta {
	var meta strategydev.IndicatorWarmupMeta
	if flow == nil || res.Data == nil {
		return meta
	}
	data := res.Data
	strategydev.StripSignalPriceAnalysisPayload(data)
	months = strategydev.CapMonthsBack(months)
	meta = strategydev.FinalizeDiagnoseProbeSeries(data, months, buySignal, sellSignal)
	if fromDiagnoseTool {
		applyDiagnoseDetailMerge(flow, res, data)
		return meta
	}
	applyProbeSeriesToFlow(flow, res, data, meta)
	return meta
}

func applyProbeSeriesToFlow(
	flow *Flow,
	res tools.Result,
	data map[string]any,
	meta strategydev.IndicatorWarmupMeta,
) {
	flow.ProbeBuyHits = probeHitCount(data, "buy_hits", "buy_merged", 1)
	flow.ProbeSellHits = probeHitCount(data, "sell_hits", "sell_merged", -1)
	if hits, ok := data["hits"].(map[string]any); ok {
		if flow.ProbeBuyHits == 0 {
			flow.ProbeBuyHits = intAny(hits["buy"])
		}
		if flow.ProbeSellHits == 0 {
			flow.ProbeSellHits = intAny(hits["sell"])
		}
	}
	flow.ProbeBarCount = intAny(data["bar_count"])
	if flow.ProbeBarCount == 0 {
		if bars, ok := data["bars"].([]any); ok {
			flow.ProbeBarCount = len(bars)
		}
	}
	flow.ProbeRaw = map[string]any{
		"bars":          data["bars"],
		"buy_merged":    data["buy_merged"],
		"sell_merged":   data["sell_merged"],
		"bars_lookback": data["bars_lookback"],
	}
	if v := data["pattern_hits"]; v != nil {
		flow.ProbeRaw["pattern_hits"] = v
	}
	attachWarmupMetaToProbeRaw(flow.ProbeRaw, meta)
	if flow.Strategies != nil && len(flow.Strategies) > 0 {
		flow.Strategies[0].BuyHits = flow.ProbeBuyHits
		flow.Strategies[0].SellHits = flow.ProbeSellHits
	}
}

func attachWarmupMetaToProbeRaw(probeRaw map[string]any, meta strategydev.IndicatorWarmupMeta) {
	if probeRaw == nil {
		return
	}
	if meta.Note != "" {
		probeRaw["indicator_warmup_note"] = meta.Note
	}
	if meta.RequiredWarmupBars > 0 {
		probeRaw["indicator_warmup_bars"] = meta.RequiredWarmupBars
	}
	if meta.PrefixBarsBeforeWindow > 0 {
		probeRaw["indicator_warmup_prefix_bars"] = meta.PrefixBarsBeforeWindow
	}
	if meta.SuppressedInWindowBars > 0 {
		probeRaw["indicator_warmup_suppressed_bars"] = meta.SuppressedInWindowBars
	}
}

// applyDiagnoseDetailMerge stores diagnose rules/verdict for build_detail skip without replacing probe hits.
func applyDiagnoseDetailMerge(flow *Flow, res tools.Result, data map[string]any) {
	if flow == nil || data == nil {
		return
	}
	flow.DiagnoseRaw = data
	flow.DiagnoseVerdict = strings.TrimSpace(fmt.Sprint(data["verdict"]))
	flow.DiagnoseSummary = strings.TrimSpace(fmt.Sprint(data["summary"]))
	if flow.DiagnoseSummary == "" {
		flow.DiagnoseSummary = strings.TrimSpace(res.Summary)
	}
	flow.DiagnoseMergedAtProbe = true
}
