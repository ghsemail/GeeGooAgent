package chat

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

// applySignalDiagnoseSeriesData normalizes probe/diagnose tool payload onto the flow.
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
	if meta.Note != "" {
		flow.ProbeRaw["indicator_warmup_note"] = meta.Note
	}
	if meta.RequiredWarmupBars > 0 {
		flow.ProbeRaw["indicator_warmup_bars"] = meta.RequiredWarmupBars
	}
	if meta.PrefixBarsBeforeWindow > 0 {
		flow.ProbeRaw["indicator_warmup_prefix_bars"] = meta.PrefixBarsBeforeWindow
	}
	if meta.SuppressedInWindowBars > 0 {
		flow.ProbeRaw["indicator_warmup_suppressed_bars"] = meta.SuppressedInWindowBars
	}
	if fromDiagnoseTool {
		flow.DiagnoseRaw = data
		flow.DiagnoseVerdict = strings.TrimSpace(fmt.Sprint(data["verdict"]))
		flow.DiagnoseSummary = strings.TrimSpace(fmt.Sprint(data["summary"]))
		if flow.DiagnoseSummary == "" {
			flow.DiagnoseSummary = strings.TrimSpace(res.Summary)
		}
		flow.DiagnoseMergedAtProbe = true
	}
	return meta
}
