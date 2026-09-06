package slots

import (
	"fmt"
	"strings"
)

// NormalizeSignalChain fills missing index/type/param on catalog buy/sell rules
// so probe/backtest schema validation passes.
func NormalizeSignalChain(rules []any) []any {
	if len(rules) == 0 {
		return rules
	}
	out := make([]any, 0, len(rules))
	for _, raw := range rules {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, normalizeSignalRule(row, ""))
	}
	return out
}

func normalizeSignalRulesPair(buy, sell []any) ([]any, []any) {
	buy = NormalizeSignalChain(buy)
	typeByIndex := signalTypesByIndex(buy)
	if len(sell) == 0 {
		return buy, cloneSignalChain(buy)
	}
	outSell := make([]any, 0, len(sell))
	for _, raw := range sell {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		idx := strings.ToUpper(strings.TrimSpace(fmt.Sprint(row["index"])))
		fallback := typeByIndex[idx]
		outSell = append(outSell, normalizeSignalRule(row, fallback))
	}
	return buy, outSell
}

func signalTypesByIndex(rules []any) map[string]string {
	out := map[string]string{}
	for _, raw := range rules {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		idx := strings.ToUpper(strings.TrimSpace(fmt.Sprint(row["index"])))
		typ := strings.TrimSpace(fmt.Sprint(row["type"]))
		if idx != "" && typ != "" {
			out[idx] = typ
		}
	}
	return out
}

func cloneSignalChain(rules []any) []any {
	out := make([]any, 0, len(rules))
	for _, raw := range rules {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		cp := map[string]any{}
		for k, v := range row {
			cp[k] = v
		}
		out = append(out, cp)
	}
	return out
}

func normalizeSignalRule(row map[string]any, fallbackType string) map[string]any {
	out := map[string]any{}
	if idx := strings.TrimSpace(fmt.Sprint(row["index"])); idx != "" {
		out["index"] = idx
	}
	if p, ok := row["param"].(map[string]any); ok && p != nil {
		out["param"] = p
	} else {
		out["param"] = map[string]any{}
	}
	typ := strings.TrimSpace(fmt.Sprint(row["type"]))
	if typ == "" {
		typ = strings.TrimSpace(fallbackType)
	}
	if typ == "" {
		idx := strings.ToUpper(strings.TrimSpace(fmt.Sprint(out["index"])))
		if idx == "NOSIGNAL" {
			typ = "signal"
		} else {
			typ = "signal"
		}
	}
	out["type"] = typ
	return out
}
