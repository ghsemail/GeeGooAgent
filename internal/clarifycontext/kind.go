package clarifycontext

import "strings"

// InferKind classifies a clarify prompt for the decision model.
func InferKind(question string, choices []string) string {
	q := strings.ToLower(strings.TrimSpace(question))
	blob := q
	for _, c := range choices {
		blob += " " + strings.ToLower(c)
	}
	switch {
	case strings.Contains(q, "先做哪一步") || strings.Contains(q, "分析") && strings.Contains(q, "回测"):
		return "compound_steps"
	case strings.Contains(blob, "测买卖点") || strings.Contains(blob, "只测点") || strings.Contains(blob, "跑回测看收益"):
		return "intent_disambiguation"
	case strings.Contains(blob, "组合信号") || strings.Contains(blob, "单指标") || strings.Contains(blob, "grid"):
		return "strategy_family"
	case strings.Contains(blob, "止盈") || strings.Contains(blob, "止损") || strings.Contains(q, "回测方案"):
		return "backtest_plan_variant"
	case strings.Contains(q, "选择市场") || strings.Contains(q, "哪一只") || strings.Contains(q, "哪支"):
		return "symbol_pick"
	default:
		return "generic"
	}
}
