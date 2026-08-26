package tools

import (
	"fmt"
	"strings"
)

// NormalizeSignalRules ensures SmartTrade backtest/probe rules include type.
// Catalog combinations often omit type on sell_signal items.
func NormalizeSignalRules(rules []any) []any {
	if len(rules) == 0 {
		return rules
	}
	out := make([]any, 0, len(rules))
	for _, raw := range rules {
		rule, ok := raw.(map[string]any)
		if !ok {
			out = append(out, raw)
			continue
		}
		clone := map[string]any{}
		for k, v := range rule {
			clone[k] = v
		}
		if ruleNeedsDefaultType(clone) {
			clone["type"] = "signal"
		}
		out = append(out, clone)
	}
	return out
}

func ruleNeedsDefaultType(rule map[string]any) bool {
	if rule == nil {
		return true
	}
	v, ok := rule["type"]
	if !ok || v == nil {
		return true
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s) == ""
	}
	return strings.TrimSpace(fmt.Sprint(v)) == ""
}

// SanitizeBacktestSignalArgs normalizes buy_signal/sell_signal on run/probe payloads.
func SanitizeBacktestSignalArgs(args map[string]any) {
	if args == nil {
		return
	}
	if buy, ok := args["buy_signal"].([]any); ok && len(buy) > 0 {
		args["buy_signal"] = NormalizeSignalRules(buy)
	}
	if sell, ok := args["sell_signal"].([]any); ok && len(sell) > 0 {
		args["sell_signal"] = NormalizeSignalRules(sell)
	}
}
