package slots

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// SignalPlan describes which strategy/signal to resolve before probe or backtest.
type SignalPlan struct {
	SignalQuery string
	SignalKind  string // combination | indicator
}

// ResolvedSignal is the normalized buy/sell rules for probe or backtest tools.
type ResolvedSignal struct {
	Buy           []any
	Sell          []any
	Frequency     string
	StrategyLabel string
}

// ResolveSignal loads combination or index signals and clarifies when needed.
func ResolveSignal(
	ctx context.Context,
	toolCtx tools.Context,
	runTool func(context.Context, tools.CallRequest, tools.Context) tools.Result,
	plan SignalPlan,
) (ResolvedSignal, error) {
	if plan.SignalKind == "indicator" || strings.TrimSpace(plan.SignalQuery) == "" {
		return resolveIndexSignal(ctx, toolCtx, runTool, plan)
	}
	res := runTool(ctx, tools.CallRequest{Name: "get_signal_combinations", Arguments: map[string]any{}}, toolCtx)
	defer emitCatalogToolDone(toolCtx, "get_signal_combinations", res, map[string]any{})
	if res.Status != tools.StatusOK {
		return ResolvedSignal{}, fmt.Errorf("get_signal_combinations 失败：%s", res.Summary)
	}
	items := CatalogItems(res.Data)
	if len(items) == 0 {
		return ResolvedSignal{}, fmt.Errorf("没有可用的组合信号")
	}
	row, pickErr := pickCombination(items, plan.SignalQuery)
	if pickErr != nil {
		if toolCtx.ClarifyFn == nil {
			return ResolvedSignal{}, pickErr
		}
		choices := make([]string, 0, minInt(len(items), 4))
		for i := 0; i < len(items) && i < 4; i++ {
			choices = append(choices, fmt.Sprint(items[i]["name"]))
		}
		NotifyClarify(toolCtx, "请选择组合信号：", choices)
		answer, ok := toolCtx.ClarifyFn(ctx, "请选择组合信号：", choices)
		if !ok {
			return ResolvedSignal{}, pickErr
		}
		for _, it := range items {
			if fmt.Sprint(it["name"]) == answer {
				row = it
				pickErr = nil
				break
			}
		}
		if pickErr != nil {
			return ResolvedSignal{}, pickErr
		}
	}
	return rowToResolved(row)
}

func resolveIndexSignal(
	ctx context.Context,
	toolCtx tools.Context,
	runTool func(context.Context, tools.CallRequest, tools.Context) tools.Result,
	plan SignalPlan,
) (ResolvedSignal, error) {
	res := runTool(ctx, tools.CallRequest{Name: "get_index_signals", Arguments: map[string]any{}}, toolCtx)
	defer emitCatalogToolDone(toolCtx, "get_index_signals", res, map[string]any{})
	if res.Status != tools.StatusOK {
		return ResolvedSignal{}, fmt.Errorf("get_index_signals 失败：%s", res.Summary)
	}
	items := CatalogItems(res.Data)
	row, pickErr := pickIndexSignal(ctx, toolCtx, items, plan.SignalQuery)
	if pickErr != nil {
		return ResolvedSignal{}, pickErr
	}
	buy, sell, frequency, label, err := indexRowToRules(row)
	if err != nil {
		return ResolvedSignal{}, err
	}
	return ResolvedSignal{Buy: buy, Sell: sell, Frequency: frequency, StrategyLabel: label}, nil
}

func pickIndexSignal(ctx context.Context, toolCtx tools.Context, items []map[string]any, query string) (map[string]any, error) {
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
	limit := minInt(len(matches), 4)
	choices := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		choices = append(choices, fmt.Sprint(matches[i]["name"]))
	}
	question := fmt.Sprintf("找到多个「%s」相关信号，请选择：", query)
	if toolCtx.ClarifyFn == nil {
		return nil, fmt.Errorf("%s %s", question, strings.Join(choices, " / "))
	}
	NotifyClarify(toolCtx, question, choices)
	answer, ok := toolCtx.ClarifyFn(ctx, question, choices)
	if !ok {
		return nil, fmt.Errorf("请选择信号")
	}
	for _, row := range matches {
		if fmt.Sprint(row["name"]) == answer {
			return row, nil
		}
	}
	return nil, fmt.Errorf("未匹配所选信号「%s」", answer)
}

func indexRowToRules(row map[string]any) (buy, sell []any, frequency, label string, err error) {
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
	label = strings.TrimSpace(fmt.Sprint(row["name"]))
	return buy, sell, frequency, label, nil
}

func rowToResolved(row map[string]any) (ResolvedSignal, error) {
	buy, _ := row["buy_signal"].([]any)
	sell, _ := row["sell_signal"].([]any)
	if len(buy) == 0 {
		return ResolvedSignal{}, fmt.Errorf("组合信号缺少 buy_signal")
	}
	if len(sell) == 0 {
		sell = buy
	}
	frequency := strings.TrimSpace(fmt.Sprint(row["frequency"]))
	if frequency == "" {
		frequency = "60m"
	}
	return ResolvedSignal{
		Buy:           buy,
		Sell:          sell,
		Frequency:     frequency,
		StrategyLabel: strings.TrimSpace(fmt.Sprint(row["name"])),
	}, nil
}

// PickCombination selects a combination signal row by token overlap.
func PickCombination(items []map[string]any, query string) (map[string]any, error) {
	return pickCombination(items, query)
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

// ApplySignalHeuristics fills signal_query/kind from user text when missing.
func ApplySignalHeuristics(plan *SignalPlan, content string) {
	if plan == nil {
		return
	}
	upper := strings.ToUpper(content)
	switch {
	case strings.Contains(upper, "SAR") && strings.Contains(upper, "MACD"):
		plan.SignalKind = "combination"
		plan.SignalQuery = "SAR MACD"
	case strings.Contains(upper, "RSI"):
		plan.SignalKind = "indicator"
		plan.SignalQuery = "RSI"
	case strings.Contains(upper, "SAR"):
		plan.SignalKind = "indicator"
		plan.SignalQuery = "SAR"
	case strings.Contains(content, "共振"):
		plan.SignalKind = "combination"
		if strings.TrimSpace(plan.SignalQuery) == "" {
			plan.SignalQuery = "共振"
		}
	case strings.Contains(content, "组合"):
		plan.SignalKind = "combination"
		if strings.TrimSpace(plan.SignalQuery) == "" {
			plan.SignalQuery = "组合"
		}
	}
}
