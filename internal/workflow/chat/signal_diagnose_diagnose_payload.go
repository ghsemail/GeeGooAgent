package chat

import (
	"fmt"
	"strings"
)

// diagnoseBarCount reads bar count from probe or diagnose tool payloads.
// diagnose_bot_signal_series often puts count under range.bar_count, not top-level bar_count/bars.
func diagnoseBarCount(data map[string]any) int {
	if data == nil {
		return 0
	}
	if n := intAny(data["bar_count"]); n > 0 {
		return n
	}
	if bars, ok := data["bars"].([]any); ok && len(bars) > 0 {
		return len(bars)
	}
	if rng, ok := data["range"].(map[string]any); ok {
		if n := intAny(rng["bar_count"]); n > 0 {
			return n
		}
	}
	return 0
}

func diagnoseRuleCount(data map[string]any) int {
	if data == nil {
		return 0
	}
	n := 0
	for _, key := range []string{"buy_rules", "sell_rules"} {
		rules, ok := data[key].([]any)
		if !ok {
			continue
		}
		n += len(rules)
	}
	return n
}

// DiagnoseDetailPayloadUsable reports whether diagnose response is safe to skip build_detail.
// Empty range + no rules must not skip (Signal returned shell OK without explainable detail).
func DiagnoseDetailPayloadUsable(data map[string]any) bool {
	if data == nil {
		return false
	}
	if diagnoseRuleCount(data) > 0 {
		return true
	}
	if strings.TrimSpace(fmt.Sprint(data["summary"])) != "" && strings.TrimSpace(fmt.Sprint(data["verdict"])) != "" {
		if diagnoseBarCount(data) > 0 {
			return true
		}
	}
	for _, key := range []string{"buy_rules", "sell_rules"} {
		rules, ok := data[key].([]any)
		if !ok {
			continue
		}
		for _, item := range rules {
			row, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if samples, ok := row["sample_reasons"].([]any); ok && len(samples) > 0 {
				return true
			}
			if lb, ok := row["last_bar"].(map[string]any); ok && len(lb) > 0 {
				return true
			}
		}
	}
	return false
}
