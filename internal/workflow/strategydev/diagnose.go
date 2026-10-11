package strategydev

import (
	"context"
	"fmt"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func runDiagnose(
	ctx context.Context,
	run ToolRunner,
	code, freq string,
	months int,
	buy, sell *Candidate,
	regime string,
	usePriceFilter bool,
) (DiagnoseSummary, *PriceFilterResult, *EpisodeTriggerFilterStats, *PriceAnalysis, map[string]any, error) {
	if buy == nil || sell == nil || isNoSignalRules(buy.Buy) {
		return DiagnoseSummary{}, nil, nil, nil, nil, fmt.Errorf("买入策略无效")
	}
	if isNoSignalRules(sell.Sell) && len(sell.Buy) > 0 {
		sell.Sell = sell.Buy
	}
	res := run(ctx, "diagnose_bot_signal_series", map[string]any{
		"code":        code,
		"frequency":   NormalizeFrequency(freq),
		"buy_signal":  buy.Buy,
		"sell_signal": coalesceSellSignal(*sell),
		"months_back": months,
		"side":        "both",
	})
	if res.Status != tools.StatusOK {
		return DiagnoseSummary{}, nil, nil, nil, nil, fmt.Errorf("diagnose 失败：%s", res.Summary)
	}
	bars, _ := res.Data["bars"].([]any)
	buyMerged, _ := res.Data["buy_merged"].([]any)
	sellMerged, _ := res.Data["sell_merged"].([]any)
	if len(bars) == 0 {
		probe := run(ctx, "probe_bot_signal_series", map[string]any{
			"code": code, "frequency": NormalizeFrequency(freq),
			"buy_signal": buy.Buy, "sell_signal": coalesceSellSignal(*sell),
			"months_back": months,
		})
		if probe.Status == tools.StatusOK {
			bars, _ = probe.Data["bars"].([]any)
			buyMerged, _ = probe.Data["buy_merged"].([]any)
			sellMerged, _ = probe.Data["sell_merged"].([]any)
		}
	}
	summary, pf, pstats := finishDiagnoseFromMerged(bars, freq, buyMerged, sellMerged, regime, res.Data, usePriceFilter)
	var pa *PriceAnalysis
	if len(bars) > 0 {
		spot := slots.FloatAny(lastBarField(bars, "close"))
		kl := KeyLevelBandsFromProbeData(res.Data)
		built := BuildRunPriceAnalysis(ctx, run, code, months, spot, kl, freq)
		pa = &built
	}
	return summary, pf, pstats, pa, res.Data, nil
}

func finishDiagnoseFromMerged(
	bars []any,
	freq string,
	buyMerged, sellMerged []any,
	regime string,
	rawData map[string]any,
	usePriceFilter bool,
) (DiagnoseSummary, *PriceFilterResult, *EpisodeTriggerFilterStats) {
	var pf *PriceFilterResult
	var pstats *EpisodeTriggerFilterStats
	if usePriceFilter && len(bars) > 0 {
		var stats EpisodeTriggerFilterStats
		var pfVal PriceFilterResult
		buyMerged, sellMerged, pfVal, stats = ApplyPriceAnalysisEpisodeFilter(bars, freq, buyMerged, sellMerged)
		pf = &pfVal
		pstats = &stats
	}
	buyHits := hitCount(nil, "buy_hits", buyMerged, 1)
	sellHits := hitCount(nil, "sell_hits", sellMerged, -1)
	_ = rawData
	eval := slots.EvaluateSignalEpisodes(bars, buyMerged, sellMerged)
	if pstats != nil {
		EnrichEpisodeFilterStatsPostEval(pstats, eval)
	}
	summary := summarizeEval(eval, regime, buyHits, sellHits)
	return summary, pf, pstats
}

func coalesceSellSignal(sell Candidate) []any {
	if len(sell.Sell) > 0 {
		return sell.Sell
	}
	return sell.Buy
}

func hitCount(data map[string]any, compactKey string, merged []any, target int) int {
	if v, ok := data[compactKey]; ok {
		return int(slots.FloatAny(v))
	}
	n := 0
	for _, raw := range merged {
		if int(slots.FloatAny(raw)) == target {
			n++
		}
	}
	return n
}

func summarizeEval(eval slots.SignalEpisodeEval, regime string, buyHits, sellHits int) DiagnoseSummary {
	primary := eval.BuyEpisodes
	primarySide := "buy"
	if regime == RegimeDown || regime == RegimeWeakDown {
		primary = eval.SellEpisodes
		primarySide = "sell"
	}
	var cum float64
	for _, d := range eval.BuyDetails {
		if d.Complete {
			cum += d.DirectionReturn
		}
	}
	for _, d := range eval.SellDetails {
		if d.Complete {
			cum += d.DirectionReturn
		}
	}
	return DiagnoseSummary{
		BuyHits:      buyHits,
		SellHits:     sellHits,
		CompleteSegs: primary.CompleteCount,
		HitRate:      primary.HitRate,
		AvgDirection: primary.AvgReturn,
		CumPoints:    cum * 100,
		PrimarySide:  primarySide,
		Eval:         eval,
	}
}

func lastBarField(bars []any, key string) any {
	if len(bars) == 0 {
		return nil
	}
	if row, ok := bars[len(bars)-1].(map[string]any); ok {
		return row[key]
	}
	return nil
}

// KeyLevelBandsFromProbeData reads support/resistance from probe/diagnose payload.
func KeyLevelBandsFromProbeData(data map[string]any) KeyLevelBands {
	if data == nil {
		return KeyLevelBands{}
	}
	raw := data["key_levels"]
	m, ok := raw.(map[string]any)
	if !ok {
		return KeyLevelBands{}
	}
	return KeyLevelBands{
		SupportLow:  slots.FloatAny(m["support_low"]),
		SupportHigh: slots.FloatAny(m["support_high"]),
		ResistLow:   slots.FloatAny(m["resist_low"]),
		ResistHigh:  slots.FloatAny(m["resist_high"]),
	}
}

func ApplyHardVerdictRules(d DiagnoseSummary) string {
	if d.CompleteSegs < 3 {
		return VerdictLongWindow
	}
	if d.AvgDirection <= 0 {
		return VerdictSwapPair
	}
	return VerdictPass
}
