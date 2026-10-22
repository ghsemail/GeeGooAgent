package chat

import (
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func cloneProbeArgs(src map[string]any) map[string]any {
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func recordProbeToolMeta(flow *Flow, toolName string, res tools.Result) {
	if flow == nil {
		return
	}
	prefix := "run_probe:" + toolName
	if res.Status != tools.StatusOK || res.Data == nil {
		recordPhaseGauge(flow, prefix+":ok", 0)
		return
	}
	recordPhaseGauge(flow, prefix+":ok", 1)
	data := res.Data
	barCount := int64(intAny(data["bar_count"]))
	if barCount == 0 {
		if bars, ok := data["bars"].([]any); ok {
			barCount = int64(len(bars))
		}
	}
	recordPhaseGauge(flow, prefix+":bar_count", barCount)
	recordPhaseGauge(flow, prefix+":buy_hits", int64(probeHitCount(data, "buy_hits", "buy_merged", 1)))
	recordPhaseGauge(flow, prefix+":sell_hits", int64(probeHitCount(data, "sell_hits", "sell_merged", -1)))
}
