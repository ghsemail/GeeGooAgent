package tools

import (
	"fmt"
	"strings"
)

var catalogFrequencyPreference = []string{"60m", "daily", "15m", "5m", "1m", "1h", "4h"}

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

// NormalizeCatalogFrequency picks one K-line period when catalog rows return a string
// or a list of supported frequencies (e.g. ["5m","60m","daily"]).
func NormalizeCatalogFrequency(raw any) string {
	const defaultFreq = "60m"
	if raw == nil {
		return defaultFreq
	}
	switch v := raw.(type) {
	case string:
		if s := strings.TrimSpace(v); s != "" {
			if strings.HasPrefix(s, "[") {
				return defaultFreq
			}
			return s
		}
	case []string:
		if picked := pickPreferredFrequency(v); picked != "" {
			return picked
		}
	case []any:
		if picked := pickPreferredFrequency(anyToStrings(v)); picked != "" {
			return picked
		}
	}
	s := strings.TrimSpace(fmt.Sprint(raw))
	if s == "" || strings.HasPrefix(s, "[") {
		return defaultFreq
	}
	return s
}

func pickPreferredFrequency(candidates []string) string {
	for _, pref := range catalogFrequencyPreference {
		for _, c := range candidates {
			if strings.EqualFold(strings.TrimSpace(c), pref) {
				return pref
			}
		}
	}
	for _, c := range candidates {
		if s := strings.TrimSpace(c); s != "" {
			return s
		}
	}
	return ""
}

func anyToStrings(items []any) []string {
	out := make([]string, 0, len(items))
	for _, raw := range items {
		if s := strings.TrimSpace(fmt.Sprint(raw)); s != "" {
			out = append(out, s)
		}
	}
	return out
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
