package playbookexec

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func (r *Router) resolveStock(
	ctx context.Context,
	toolCtx tools.Context,
	plan BacktestRunPlan,
	recordTool func(name, status, summary string),
) (code, name, market string, err error) {
	res := r.runTool(ctx, toolCtx, "search_code", map[string]any{"regex": plan.StockQuery}, recordTool)
	if res.Status != tools.StatusOK {
		return "", "", "", fmt.Errorf("search_code 失败：%s", res.Summary)
	}
	items := catalogItems(res.Data)
	if len(items) == 0 {
		return "", "", "", fmt.Errorf("未找到标的「%s」，请换名称或代码", plan.StockQuery)
	}
	row := items[0]
	if len(items) > 1 {
		if picked, ok := pickStockRow(items, plan.StockQuery); ok {
			row = picked
		} else if toolCtx.ClarifyFn != nil {
			choices := make([]string, 0, minInt(len(items), 4))
			for i := 0; i < len(items) && i < 4; i++ {
				it := items[i]
				choices = append(choices, fmt.Sprintf("%v %v", it["code"], it["name"]))
			}
			answer, ok := toolCtx.ClarifyFn(ctx, "找到多个标的，请选择：", choices)
			if ok {
				for _, it := range items {
					label := fmt.Sprintf("%v %v", it["code"], it["name"])
					if label == answer || strings.Contains(answer, fmt.Sprint(it["code"])) {
						row = it
						break
					}
				}
			}
		}
	}
	code = strings.TrimSpace(fmt.Sprint(row["code"]))
	name = strings.TrimSpace(fmt.Sprint(row["name"]))
	market = strings.TrimSpace(fmt.Sprint(row["market"]))
	if code == "" {
		return "", "", "", fmt.Errorf("search_code 未返回有效 code")
	}
	return code, name, market, nil
}

type resolvedSignals struct {
	Buy           []any
	Sell          []any
	Frequency     string
	StrategyLabel string
	SignalID      string
}

func (r *Router) resolveSignals(
	ctx context.Context,
	toolCtx tools.Context,
	plan BacktestRunPlan,
	recordTool func(name, status, summary string),
) (resolvedSignals, error) {
	if plan.SignalKind == "indicator" || strings.TrimSpace(plan.SignalQuery) == "" {
		return r.resolveIndexSignal(ctx, toolCtx, plan, recordTool)
	}
	res := r.runTool(ctx, toolCtx, "get_signal_combinations", map[string]any{}, recordTool)
	if res.Status != tools.StatusOK {
		return resolvedSignals{}, fmt.Errorf("get_signal_combinations 失败：%s", res.Summary)
	}
	items := catalogItems(res.Data)
	if len(items) == 0 {
		return resolvedSignals{}, fmt.Errorf("没有可用的组合信号")
	}
	row, pickErr := pickCombination(items, plan.SignalQuery)
	if pickErr != nil {
		if toolCtx.ClarifyFn == nil {
			return resolvedSignals{}, pickErr
		}
		candidates := items
		if tokens := signalTokens(plan.SignalQuery); len(tokens) > 0 {
			var matched []map[string]any
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
					matched = append(matched, row)
				}
			}
			if len(matched) > 0 {
				candidates = matched
			}
		}
		choices := make([]string, 0, minInt(len(candidates), 4))
		for i := 0; i < len(candidates) && i < 4; i++ {
			choices = append(choices, fmt.Sprint(candidates[i]["name"]))
		}
		answer, ok := toolCtx.ClarifyFn(ctx, "请选择组合信号：", choices)
		if !ok {
			return resolvedSignals{}, pickErr
		}
		for _, it := range items {
			if fmt.Sprint(it["name"]) == answer {
				row = it
				pickErr = nil
				break
			}
		}
		if pickErr != nil {
			return resolvedSignals{}, pickErr
		}
	}
	buy, _ := row["buy_signal"].([]any)
	sell, _ := row["sell_signal"].([]any)
	if len(buy) == 0 {
		return resolvedSignals{}, fmt.Errorf("组合信号缺少 buy_signal")
	}
	buy = tools.NormalizeSignalRules(buy)
	if len(sell) == 0 {
		sell = tools.NormalizeSignalRules(buy)
	} else {
		sell = tools.NormalizeSignalRules(sell)
	}
	return resolvedSignals{
		Buy:           buy,
		Sell:          sell,
		Frequency:     tools.NormalizeCatalogFrequency(row["frequency"]),
		StrategyLabel: strings.TrimSpace(fmt.Sprint(row["name"])),
		SignalID:      signalIDFromRow(row),
	}, nil
}

func (r *Router) resolveIndexSignal(
	ctx context.Context,
	toolCtx tools.Context,
	plan BacktestRunPlan,
	recordTool func(name, status, summary string),
) (resolvedSignals, error) {
	res := r.runTool(ctx, toolCtx, "get_index_signals", map[string]any{}, recordTool)
	if res.Status != tools.StatusOK {
		return resolvedSignals{}, fmt.Errorf("get_index_signals 失败：%s", res.Summary)
	}
	items := catalogItems(res.Data)
	row, pickErr := pickIndexSignal(items, plan.SignalQuery)
	if pickErr != nil {
		return resolvedSignals{}, pickErr
	}
	var buy []any
	// Index signals need probe-style rules; fetch full combination-like shape from index row if present.
	if rawBuy, ok := row["buy_signal"].([]any); ok && len(rawBuy) > 0 {
		buy = tools.NormalizeSignalRules(rawBuy)
	} else {
		idx := strings.TrimSpace(fmt.Sprint(row["index"]))
		if idx == "" {
			return resolvedSignals{}, fmt.Errorf("单指标信号缺少 index")
		}
		buy = []any{map[string]any{"index": idx, "type": "signal", "param": row["param"]}}
	}
	sell := buy
	return resolvedSignals{
		Buy:           buy,
		Sell:          sell,
		Frequency:     tools.NormalizeCatalogFrequency(row["frequency"]),
		StrategyLabel: strings.TrimSpace(fmt.Sprint(row["name"])),
		SignalID:      signalIDFromRow(row),
	}, nil
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

func pickStockRow(items []map[string]any, query string) (map[string]any, bool) {
	q := strings.ToUpper(strings.TrimSpace(query))
	if q == "" || len(items) == 0 {
		return nil, false
	}
	qDigits := strings.TrimLeft(q, "0")
	var containsMatches []map[string]any
	for _, row := range items {
		code := strings.ToUpper(strings.TrimSpace(fmt.Sprint(row["code"])))
		if code == "" {
			continue
		}
		if code == q {
			return row, true
		}
		codeBase := strings.TrimSuffix(code, ".HK")
		codeBase = strings.TrimSuffix(codeBase, ".SH")
		codeBase = strings.TrimSuffix(codeBase, ".SZ")
		codeBase = strings.TrimSuffix(codeBase, ".US")
		if strings.HasSuffix(q, ".HK") || strings.HasSuffix(q, ".SH") || strings.HasSuffix(q, ".SZ") || strings.HasSuffix(q, ".US") {
			if code == q || strings.HasPrefix(code, q) {
				return row, true
			}
		}
		if qDigits != "" && (codeBase == qDigits || strings.HasSuffix(codeBase, qDigits) || strings.HasSuffix(qDigits, strings.TrimLeft(codeBase, "0"))) {
			return row, true
		}
		if strings.Contains(code, q) || strings.Contains(q, codeBase) {
			containsMatches = append(containsMatches, row)
		}
	}
	if len(containsMatches) == 1 {
		return containsMatches[0], true
	}
	return nil, false
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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
