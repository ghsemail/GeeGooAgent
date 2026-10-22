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

// probeArgsForDiagnoseTool strips probe-only fields; diagnose ignores bars/key_levels extras.
func probeArgsForDiagnoseTool(probeArgs map[string]any) map[string]any {
	out := cloneProbeArgs(probeArgs)
	delete(out, "include_key_levels")
	delete(out, "key_levels_mode")
	delete(out, "key_levels_align")
	out["side"] = "both"
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
	barCount := int64(diagnoseBarCount(data))
	recordPhaseGauge(flow, prefix+":bar_count", barCount)
	recordPhaseGauge(flow, prefix+":rule_count", int64(diagnoseRuleCount(data)))
}

// recordProbeHitsMeta records post-finalize (in-window) vs Signal-API raw hits on the flow.
func recordProbeHitsMeta(flow *Flow) {
	if flow == nil {
		return
	}
	prefix := "run_probe:probe_bot_signal_series"
	recordPhaseGauge(flow, prefix+":buy_hits_in_window", int64(flow.ProbeBuyHits))
	recordPhaseGauge(flow, prefix+":sell_hits_in_window", int64(flow.ProbeSellHits))
	recordPhaseGauge(flow, prefix+":buy_hits_signal_api", int64(flow.ProbeBuyHitsSignalAPI))
	recordPhaseGauge(flow, prefix+":sell_hits_signal_api", int64(flow.ProbeSellHitsSignalAPI))
}
