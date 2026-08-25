package tools

import (
	"fmt"
	"strings"
)

// compactHTTPPayload trims large catalog list responses before they reach the LLM.
func compactHTTPPayload(ctx Context, toolName string, payload any) any {
	if ctx.FullCatalogPayload {
		return payload
	}
	switch toolName {
	case "get_signal_combinations", "get_index_signals", "get_custom_signal_for_skill":
		if items, ok := payload.([]any); ok {
			return compactCatalogItems(toolName, items)
		}
	case "probe_bot_signal_series":
		if row, ok := payload.(map[string]any); ok {
			return compactProbeSeries(row)
		}
	}
	return payload
}

func compactProbeSeries(v map[string]any) map[string]any {
	out := map[string]any{
		"code":        v["code"],
		"frequency":   v["frequency"],
		"months_back": v["months_back"],
		"buy_hits":    countMergedSignals(v["buy_merged"], 1),
		"sell_hits":   countMergedSignals(v["sell_merged"], -1),
	}
	if bars, ok := v["bars"].([]any); ok {
		out["bar_count"] = len(bars)
		if len(bars) > 0 {
			if first, ok := bars[0].(map[string]any); ok {
				out["range_start"] = first["time"]
			}
			if last, ok := bars[len(bars)-1].(map[string]any); ok {
				out["range_end"] = last["time"]
			}
		}
		out["recent_buy_times"] = recentSignalTimes(bars, v["buy_merged"], 1, 3)
		out["recent_sell_times"] = recentSignalTimes(bars, v["sell_merged"], -1, 3)
	}
	return out
}

func recentSignalTimes(bars []any, mergedRaw any, target, maxN int) []string {
	merged, ok := mergedRaw.([]any)
	if !ok || len(merged) == 0 || maxN <= 0 {
		return nil
	}
	limit := len(merged)
	if len(bars) < limit {
		limit = len(bars)
	}
	out := make([]string, 0, maxN)
	for i := limit - 1; i >= 0 && len(out) < maxN; i-- {
		hit := false
		switch v := merged[i].(type) {
		case float64:
			hit = int(v) == target
		case int:
			hit = v == target
		}
		if !hit {
			continue
		}
		bar, ok := bars[i].(map[string]any)
		if !ok {
			continue
		}
		if ts, ok := bar["time"].(string); ok && ts != "" {
			out = append(out, ts)
		}
	}
	return out
}

func compactCatalogItems(toolName string, items []any) []any {
	out := make([]any, 0, len(items))
	for _, raw := range items {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		switch toolName {
		case "get_signal_combinations":
			out = append(out, compactCombinationRow(row))
		case "get_index_signals":
			out = append(out, compactIndexSignalRow(row))
		case "get_custom_signal_for_skill":
			out = append(out, compactCustomSignalRow(row))
		}
	}
	return out
}

func compactCombinationRow(row map[string]any) map[string]any {
	out := map[string]any{
		"signal_id": row["signal_id"],
		"name":      row["name"],
		"brief":     truncateText(row["brief"], 160),
	}
	if v := row["frequency"]; v != nil {
		out["frequency"] = v
	}
	if buy, ok := row["buy_signal"].([]any); ok {
		out["buy_rule_count"] = len(buy)
		out["buy_indexes"] = ruleIndexes(buy)
	}
	if sell, ok := row["sell_signal"].([]any); ok {
		out["sell_rule_count"] = len(sell)
		out["sell_indexes"] = ruleIndexes(sell)
	}
	return out
}

func compactIndexSignalRow(row map[string]any) map[string]any {
	out := map[string]any{
		"signal_id": row["signal_id"],
		"name":      row["name"],
		"brief":     truncateText(row["brief"], 160),
		"index":     row["index"],
		"frequency": row["frequency"],
	}
	return out
}

func compactCustomSignalRow(row map[string]any) map[string]any {
	out := map[string]any{
		"signal_id":             row["signal_id"],
		"name":                  row["name"],
		"brief":                 truncateText(row["brief"], 160),
		"supported_frequencies": row["supported_frequencies"],
	}
	if custom, ok := row["custom"].(map[string]any); ok {
		compactCustom := map[string]any{
			"index": custom["index"],
			"type":  custom["type"],
		}
		if param, ok := custom["param"].(map[string]any); ok {
			compactCustom["param"] = param
		}
		out["custom"] = compactCustom
	}
	return out
}

func ruleIndexes(rules []any) string {
	parts := make([]string, 0, len(rules))
	for _, raw := range rules {
		rule, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		idx := strings.TrimSpace(fmt.Sprint(rule["index"]))
		if idx != "" {
			parts = append(parts, idx)
		}
	}
	return strings.Join(parts, "+")
}

func truncateText(v any, max int) string {
	s := strings.TrimSpace(fmt.Sprint(v))
	if max <= 0 || len([]rune(s)) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max]) + "…"
}

func summarizeCatalogList(toolName string, items []any) string {
	switch toolName {
	case "get_signal_combinations":
		if len(items) == 0 {
			return "get_signal_combinations: 0 combination(s)"
		}
		first, _ := items[0].(map[string]any)
		return fmt.Sprintf("get_signal_combinations: %d combination(s); first=%v id=%v",
			len(items), first["name"], first["signal_id"])
	case "get_index_signals":
		if len(items) == 0 {
			return "get_index_signals: 0 signal(s)"
		}
		first, _ := items[0].(map[string]any)
		return fmt.Sprintf("get_index_signals: %d signal(s); first=%v index=%v",
			len(items), first["name"], first["index"])
	case "get_custom_signal_for_skill":
		if len(items) == 0 {
			return "get_custom_signal_for_skill: 0 custom strategy(ies)"
		}
		first, _ := items[0].(map[string]any)
		idx := ""
		if custom, ok := first["custom"].(map[string]any); ok {
			idx = fmt.Sprint(custom["index"])
		}
		return fmt.Sprintf("get_custom_signal_for_skill: %d strategy(ies); first=%v index=%s",
			len(items), first["name"], idx)
	default:
		return fmt.Sprintf("%s: %d item(s)", toolName, len(items))
	}
}
