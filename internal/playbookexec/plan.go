package playbookexec

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// BacktestRunPlan is the structured plan extracted before deterministic execution.
type BacktestRunPlan struct {
	StockQuery  string  `json:"stock_query"`
	SignalQuery string  `json:"signal_query"`
	SignalKind  string  `json:"signal_kind"` // combination | indicator
	Period      string  `json:"period"`
	MonthsBack  int     `json:"months_back"`
	Fund        float64 `json:"fund"`
}

const backtestPlanSystem = `你是 GeeGoo 策略回测计划解析器。只输出一个 JSON 对象，不要 markdown，不要解释。
字段：
- stock_query: 用户提到的标的名称或代码片段（如 小米、AAPL、00700）
- signal_query: 信号描述（如 SAR+MACD、RSI阈值）。必须是短标签，禁止把整句用户话填进来。
- signal_kind: combination 或 indicator
- period: 1m/2m/3m，默认 3m
- months_back: 整数，默认 3
- fund: 初始资金，默认 100000
当前消息未指定的字段：沿用会话里已出现的标的/策略名；没有就留空或用默认值。不要编造新的信号。`

var (
	reMonths = regexp.MustCompile(`(?i)(\d+)\s*(个)?月`)
	reFund   = regexp.MustCompile(`(?i)(?:资金|本金|fund)?\s*(\d+)\s*(?:万|w)?`)
)

func (r *Router) buildBacktestPlan(ctx context.Context, in Input, step int) (BacktestRunPlan, string, error) {
	plan := heuristicBacktestPlan(in.UserText)
	enrichPlanFromSession(&plan, in.Session)
	if heuristicSlotsReady(plan) {
		return plan, fmt.Sprintf("playbookexec plan(heuristic): stock=%q signal=%q kind=%s",
			plan.StockQuery, plan.SignalQuery, plan.SignalKind), nil
	}
	if r == nil || r.Gateway == nil {
		if strings.TrimSpace(plan.StockQuery) == "" {
			return plan, "", fmt.Errorf("缺少标的，请说明要回测哪只股票")
		}
		return plan, "playbookexec plan(heuristic partial)", nil
	}
	planUser := in.UserText
	if in.Session != nil {
		if ctxText := recentSessionContext(in.Session, 6); ctxText != "" {
			planUser = "会话上下文：\n" + ctxText + "\n\n当前用户消息：\n" + in.UserText
		}
	}
	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: backtestPlanSystem},
		{Role: llm.RoleUser, Content: planUser},
	}
	callCtx := llm.WithCallMeta(ctx, llm.CallMeta{Kind: llm.TaskChat, ToolSchemaCount: 0})
	resp, err := r.Gateway.ChatStream(callCtx, messages, nil, in.Session.ID, step, nil)
	if err != nil {
		return plan, "", err
	}
	parsed, err := parseBacktestPlanJSON(resp.Content)
	if err != nil {
		return heuristicBacktestPlan(in.UserText), "playbookexec plan(heuristic fallback)", nil
	}
	mergeBacktestPlan(&plan, parsed)
	if strings.TrimSpace(plan.StockQuery) == "" {
		return plan, "", fmt.Errorf("缺少标的，请说明要回测哪只股票")
	}
	return plan, fmt.Sprintf("playbookexec plan(llm): stock=%q signal=%q kind=%s",
		plan.StockQuery, plan.SignalQuery, plan.SignalKind), nil
}

func heuristicBacktestPlan(message string) BacktestRunPlan {
	plan := BacktestRunPlan{
		Period:     "3m",
		MonthsBack: 3,
		Fund:       100000,
	}
	msg := strings.TrimSpace(message)

	if m := reMonths.FindStringSubmatch(msg); len(m) > 1 {
		if n, err := parseInt(m[1]); err == nil && n > 0 {
			plan.MonthsBack = n
			plan.Period = fmt.Sprintf("%dm", n)
		}
	}
	if m := reFund.FindStringSubmatch(msg); len(m) > 1 {
		if n, err := parseInt(m[1]); err == nil && n > 0 {
			if strings.Contains(msg, "万") {
				plan.Fund = float64(n * 10000)
			} else {
				plan.Fund = float64(n)
			}
		}
	}

	plan.StockQuery = slots.ExtractStockQuery(msg)
	applySignalHeuristics(&plan, msg)
	return plan
}

func heuristicSlotsReady(plan BacktestRunPlan) bool {
	return slots.LooksLikeStockQuery(plan.StockQuery) && strings.TrimSpace(plan.SignalQuery) != ""
}

func parseBacktestPlanJSON(raw string) (BacktestRunPlan, error) {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, "{"); i >= 0 {
		if j := strings.LastIndex(raw, "}"); j > i {
			raw = raw[i : j+1]
		}
	}
	var plan BacktestRunPlan
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		return BacktestRunPlan{}, err
	}
	normalizeBacktestPlan(&plan)
	return plan, nil
}

func mergeBacktestPlan(dst *BacktestRunPlan, src BacktestRunPlan) {
	if strings.TrimSpace(src.StockQuery) != "" {
		dst.StockQuery = strings.TrimSpace(src.StockQuery)
	}
	if strings.TrimSpace(src.SignalQuery) != "" {
		dst.SignalQuery = strings.TrimSpace(src.SignalQuery)
	}
	if strings.TrimSpace(src.SignalKind) != "" {
		dst.SignalKind = strings.TrimSpace(src.SignalKind)
	}
	if strings.TrimSpace(src.Period) != "" {
		dst.Period = strings.TrimSpace(src.Period)
	}
	if src.MonthsBack > 0 {
		dst.MonthsBack = src.MonthsBack
	}
	if src.Fund > 0 {
		dst.Fund = src.Fund
	}
	normalizeBacktestPlan(dst)
}

func normalizeBacktestPlan(plan *BacktestRunPlan) {
	if plan.MonthsBack <= 0 {
		plan.MonthsBack = 3
	}
	if strings.TrimSpace(plan.Period) == "" {
		plan.Period = fmt.Sprintf("%dm", plan.MonthsBack)
	}
	if plan.Fund <= 0 {
		plan.Fund = 100000
	}
	if strings.TrimSpace(plan.SignalKind) == "" {
		if strings.Contains(strings.ToUpper(plan.SignalQuery), "SAR") &&
			strings.Contains(strings.ToUpper(plan.SignalQuery), "MACD") {
			plan.SignalKind = "combination"
		} else {
			plan.SignalKind = "indicator"
		}
	}
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &n)
	return n, err
}
