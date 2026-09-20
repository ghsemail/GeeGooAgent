package chat

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func flowHasProbePanelOverrides(flow *Flow) bool {
	if flow == nil {
		return false
	}
	return len(flow.ProbeBuyOverride) > 0 || len(flow.ProbeSellOverride) > 0
}

func resolvedSignalFromProbeOverrides(flow *Flow) slots.ResolvedSignal {
	base := slots.ResolvedSignal{Frequency: "60m", StrategyLabel: probePanelDisplayLabel(flow)}
	out := slots.MergeProbeRuleOverrides(base, flow.ProbeBuyOverride, flow.ProbeSellOverride)
	if f := strings.TrimSpace(flow.ProbeFrequencyOverride); f != "" {
		out.Frequency = f
	}
	out.StrategyLabel = probePanelDisplayLabel(flow)
	return out
}

func probePanelDisplayLabel(flow *Flow) string {
	if flow == nil {
		return ""
	}
	if label := strings.TrimSpace(flow.ProbePanelLabel); label != "" {
		return label
	}
	buy := formatProbeRuleChain(flow.ProbeBuyOverride)
	sell := formatProbeRuleChain(flow.ProbeSellOverride)
	switch {
	case buy != "" && sell != "" && buy != sell:
		return fmt.Sprintf("买入: %s · 卖出: %s", buy, sell)
	case buy != "":
		return buy
	case sell != "":
		return sell
	default:
		return ""
	}
}

func formatProbeRuleChain(rules []any) string {
	if len(rules) == 0 {
		return ""
	}
	parts := make([]string, 0, len(rules))
	for _, raw := range rules {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		idx := strings.TrimSpace(fmt.Sprint(row["index"]))
		if idx == "" {
			continue
		}
		typ := strings.TrimSpace(fmt.Sprint(row["type"]))
		if typ != "" && typ != "signal" {
			parts = append(parts, fmt.Sprintf("%s·%s", idx, typ))
		} else {
			parts = append(parts, idx)
		}
	}
	return strings.Join(parts, "+")
}

func appendProbePanelConfigLines(b *strings.Builder, flow *Flow) {
	if !flowHasProbePanelOverrides(flow) {
		return
	}
	label := probePanelDisplayLabel(flow)
	if label != "" {
		fmt.Fprintf(b, "- 策略配置：%s\n", label)
	}
	buy := formatProbeRuleChain(flow.ProbeBuyOverride)
	sell := formatProbeRuleChain(flow.ProbeSellOverride)
	if buy != "" {
		fmt.Fprintf(b, "- 买入规则：%s\n", buy)
	}
	if sell != "" && sell != buy {
		fmt.Fprintf(b, "- 卖出规则：%s\n", sell)
	}
}
