package runtimeapi

func (h *Handler) buildContextProfilesSummary(userID string) map[string]any {
	summary := h.scopedPreferencesSummary(userID)
	mergedBytes := 0
	for _, p := range h.profileStatusEntries(userID, nil) {
		if n, ok := p["bytes"].(int); ok {
			mergedBytes += n
		}
	}
	paths := make([]string, 0)
	for _, p := range h.profileStatusEntries(userID, nil) {
		if p["missing"] == true {
			continue
		}
		if path, ok := p["path"].(string); ok && path != "" {
			paths = append(paths, path)
		}
	}
	summary["merged_bytes"] = mergedBytes
	summary["truncated"] = false
	summary["paths"] = paths
	return summary
}
