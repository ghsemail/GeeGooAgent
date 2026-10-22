package chat

// diagnoseStructuredForCard exposes buy/sell rules for Dock (parity with pre-#60 card fields).
func diagnoseStructuredForCard(raw map[string]any) map[string]any {
	if raw == nil || !DiagnoseDetailPayloadUsable(raw) {
		return nil
	}
	raw = normalizeDiagnoseDetailForCard(raw)
	out := map[string]any{}
	if v := raw["verdict"]; v != nil {
		out["verdict"] = v
	}
	if v := raw["summary"]; v != nil {
		out["summary"] = v
	}
	if v := raw["hits"]; v != nil {
		out["hits"] = v
	}
	if rng := raw["range"]; rng != nil {
		out["range"] = rng
	}
	if rules, ok := raw["buy_rules"].([]any); ok && len(rules) > 0 {
		out["buy_rules"] = rules
	}
	if rules, ok := raw["sell_rules"].([]any); ok && len(rules) > 0 {
		out["sell_rules"] = rules
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
