package playbookexec

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func (r *Router) resolveStock(
	ctx context.Context,
	toolCtx tools.Context,
	plan BacktestRunPlan,
	recordTool func(name, status, summary string),
) (code, name, market string, err error) {
	runTool := func(ctx context.Context, req tools.CallRequest, tc tools.Context) tools.Result {
		return r.runTool(ctx, tc, req.Name, req.Arguments, recordTool)
	}
	return slots.ResolveStock(ctx, toolCtx, runTool, plan.StockQuery)
}

func (r *Router) resolveSignals(
	ctx context.Context,
	toolCtx tools.Context,
	plan BacktestRunPlan,
	recordTool func(name, status, summary string),
) (buy, sell []any, frequency, strategyLabel string, err error) {
	if plan.SignalKind == "indicator" || strings.TrimSpace(plan.SignalQuery) == "" {
		return r.resolveIndexSignal(ctx, toolCtx, plan, recordTool)
	}
	res := r.runTool(ctx, toolCtx, "get_signal_combinations", map[string]any{}, recordTool)
	if res.Status != tools.StatusOK {
		return nil, nil, "", "", fmt.Errorf("get_signal_combinations 失败：%s", res.Summary)
	}
	items := catalogItems(res.Data)
	if len(items) == 0 {
		return nil, nil, "", "", fmt.Errorf("没有可用的组合信号")
	}
	row, pickErr := pickCombination(items, plan.SignalQuery)
	if pickErr != nil {
		if toolCtx.ClarifyFn == nil {
			return nil, nil, "", "", pickErr
		}
		choices := make([]string, 0, minInt(len(items), 4))
		for i := 0; i < len(items) && i < 4; i++ {
			choices = append(choices, fmt.Sprint(items[i]["name"]))
		}
		answer, ok := toolCtx.ClarifyFn(ctx, "请选择组合信号：", choices)
		if !ok {
			return nil, nil, "", "", pickErr
		}
		for _, it := range items {
			if fmt.Sprint(it["name"]) == answer {
				row = it
				pickErr = nil
				break
			}
		}
		if pickErr != nil {
			return nil, nil, "", "", pickErr
		}
	}
	buy, _ = row["buy_signal"].([]any)
	sell, _ = row["sell_signal"].([]any)
	if len(buy) == 0 {
		return nil, nil, "", "", fmt.Errorf("组合信号缺少 buy_signal")
	}
	if len(sell) == 0 {
		sell = buy
	}
	frequency = strings.TrimSpace(fmt.Sprint(row["frequency"]))
	if frequency == "" {
		frequency = "60m"
	}
	strategyLabel = strings.TrimSpace(fmt.Sprint(row["name"]))
	return buy, sell, frequency, strategyLabel, nil
}

func (r *Router) resolveIndexSignal(
	ctx context.Context,
	toolCtx tools.Context,
	plan BacktestRunPlan,
	recordTool func(name, status, summary string),
) (buy, sell []any, frequency, strategyLabel string, err error) {
	res := r.runTool(ctx, toolCtx, "get_index_signals", map[string]any{}, recordTool)
	if res.Status != tools.StatusOK {
		return nil, nil, "", "", fmt.Errorf("get_index_signals 失败：%s", res.Summary)
	}
	items := catalogItems(res.Data)
	row, pickErr := pickIndexSignal(items, plan.SignalQuery)
	if pickErr != nil {
		return nil, nil, "", "", pickErr
	}
	// Index signals need probe-style rules; fetch full combination-like shape from index row if present.
	if rawBuy, ok := row["buy_signal"].([]any); ok && len(rawBuy) > 0 {
		buy = rawBuy
	} else {
		idx := strings.TrimSpace(fmt.Sprint(row["index"]))
		if idx == "" {
			return nil, nil, "", "", fmt.Errorf("单指标信号缺少 index")
		}
		buy = []any{map[string]any{"index": idx, "type": "signal", "param": row["param"]}}
	}
	sell = buy
	frequency = strings.TrimSpace(fmt.Sprint(row["frequency"]))
	if frequency == "" {
		frequency = "60m"
	}
	strategyLabel = strings.TrimSpace(fmt.Sprint(row["name"]))
	return buy, sell, frequency, strategyLabel, nil
}

func catalogItems(data map[string]any) []map[string]any {
	if data == nil {
		return nil
	}
	if items, ok := data["items"].([]any); ok {
		return mapsFromAny(items)
	}
	if arr, ok := data["data"].([]any); ok {
		return mapsFromAny(arr)
	}
	if arr, ok := data["value"].([]any); ok {
		return mapsFromAny(arr)
	}
	return mapsFromAny([]any{data})
}

func mapsFromAny(items []any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, raw := range items {
		row, ok := raw.(map[string]any)
		if ok {
			out = append(out, row)
		}
	}
	return out
}

func pickCombination(items []map[string]any, query string) (map[string]any, error) {
	tokens := signalTokens(query)
	if len(tokens) == 0 {
		if len(items) == 1 {
			return items[0], nil
		}
		return nil, fmt.Errorf("请说明要用哪种组合信号")
	}
	var matches []map[string]any
	for _, row := range items {
		name := strings.ToUpper(fmt.Sprint(row["name"]))
		ok := true
		for _, tok := range tokens {
			if !strings.Contains(name, tok) {
				ok = false
				break
			}
		}
		if ok {
			matches = append(matches, row)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) == 0 && len(items) == 1 {
		return items[0], nil
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("未找到匹配「%s」的组合信号", query)
	}
	return bestCombinationMatch(matches, tokens), nil
}

func bestCombinationMatch(matches []map[string]any, tokens []string) map[string]any {
	best := matches[0]
	bestScore := scoreCombinationMatch(fmt.Sprint(best["name"]), tokens)
	for _, row := range matches[1:] {
		if score := scoreCombinationMatch(fmt.Sprint(row["name"]), tokens); score > bestScore {
			best = row
			bestScore = score
		}
	}
	return best
}

func scoreCombinationMatch(name string, tokens []string) int {
	upper := strings.ToUpper(name)
	score := 0
	prevPos := -1
	for i, tok := range tokens {
		pos := strings.Index(upper, tok)
		if pos < 0 {
			return -1
		}
		if i == 0 && pos == 0 {
			score += 30
		}
		if prevPos >= 0 {
			if pos >= prevPos {
				score += 100 - (pos-prevPos)/10
			} else {
				score -= 50
			}
		}
		prevPos = pos
	}
	score -= len([]rune(name)) / 10
	return score
}

func pickIndexSignal(items []map[string]any, query string) (map[string]any, error) {
	q := strings.ToUpper(strings.TrimSpace(query))
	if q == "" && len(items) > 0 {
		return items[0], nil
	}
	var matches []map[string]any
	for _, row := range items {
		name := strings.ToUpper(fmt.Sprint(row["name"]))
		index := strings.ToUpper(fmt.Sprint(row["index"]))
		if strings.Contains(name, q) || strings.Contains(index, q) {
			matches = append(matches, row)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("未找到匹配「%s」的单指标信号", query)
	}
	return nil, fmt.Errorf("找到多个单指标信号，需要 clarify")
}

func signalTokens(query string) []string {
	parts := strings.Fields(strings.ToUpper(strings.NewReplacer("+", " ", "，", " ", "、", " ", "和", " ").Replace(query)))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || p == "信号" || p == "趋势" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func formatBacktestReply(code, name, strategyLabel string, data map[string]any) string {
	profit := fmt.Sprint(data["profit_rate"])
	finalValue := fmt.Sprint(data["final_value"])
	logID := fmt.Sprint(data["log_id"])
	trades := fmt.Sprint(data["trade_count"])
	drawdown := fmt.Sprint(data["drawdown"])
	return fmt.Sprintf("## %s %s · SmartTrade 回测\n\n- **策略**：%s\n- **收益率**：%s%%\n- **最终资产**：%s\n- **最大回撤**：%s%%\n- **成交笔数**：%s\n- **log_id**：%s\n\n> 由 playbook 确定性执行（run_strategy_backtest）",
		name, code, strategyLabel, profit, finalValue, drawdown, trades, logID)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
