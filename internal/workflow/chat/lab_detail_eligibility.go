package chat

// flowHasProbeBars reports whether the flow carries OHLCV bars for lazy lab-detail.
func flowHasProbeBars(flow *Flow) bool {
	if flow == nil {
		return false
	}
	if bars, _ := flow.ProbeRaw["bars"].([]any); len(bars) > 0 {
		return true
	}
	return false
}

// FlowSupportsLabDetail is true when lab-detail can run (Persist Result.lab path or archived runner flow).
func FlowSupportsLabDetail(flow *Flow) bool {
	if flow == nil {
		return false
	}
	if flowHasProbeBars(flow) {
		return true
	}
	return len(flow.LabEpisodeSummaries) > 0
}
