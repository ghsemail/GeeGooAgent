package chat

import "time"

func recordPhaseTiming(flow *Flow, phase string, d time.Duration) {
	if flow == nil || phase == "" || d <= 0 {
		return
	}
	if flow.PhaseTimingsMs == nil {
		flow.PhaseTimingsMs = map[string]int64{}
	}
	flow.PhaseTimingsMs[phase] += d.Milliseconds()
}

func recordPhaseGauge(flow *Flow, key string, value int64) {
	if flow == nil || key == "" {
		return
	}
	if flow.PhaseTimingsMs == nil {
		flow.PhaseTimingsMs = map[string]int64{}
	}
	flow.PhaseTimingsMs[key] = value
}

func recordWorkflowTimingTotal(flow *Flow) {
	if flow == nil || flow.WorkflowStartedAt.IsZero() {
		return
	}
	if flow.PhaseTimingsMs == nil {
		flow.PhaseTimingsMs = map[string]int64{}
	}
	flow.PhaseTimingsMs["_total"] = time.Since(flow.WorkflowStartedAt).Milliseconds()
}
