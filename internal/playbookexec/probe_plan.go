package playbookexec

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// ProbeRunPlan is the structured plan extracted before deterministic signal probe execution.
type ProbeRunPlan struct {
	StockQuery  string `json:"stock_query"`
	SignalQuery string `json:"signal_query"`
	SignalKind  string `json:"signal_kind"`
	MonthsBack  int    `json:"months_back"`
	Frequency   string `json:"frequency"`
}

const defaultProbeCombinationQuery = "SAR MACD"

const probePlanSystem = `你是 GeeGoo 信号探测（买卖点）计划解析器。只输出一个 JSON 对象，不要 markdown，不要解释。
字段：
- stock_query: 用户要探测的标的名称或代码（如 中际旭创、00700、AAPL）。禁止填「买卖点」「卖点」「买点」等意图词。
- signal_query: 信号描述（如 SAR+MACD、RSI阈值）。短标签即可，禁止把整句用户话填进来。
- signal_kind: combination 或 indicator
- months_back: 整数，默认 3
- frequency: 可选，如 60m、15m；未指定留空
当前消息未指定的字段：沿用会话里已出现的标的/策略名；没有就留空。用户只问「有没有买卖点」且未指定策略时，signal_query 可留空（执行层会用 SAR+MACD 组合默认）。不要编造新的信号名。`

func (r *Router) buildProbePlan(ctx context.Context, in Input, step int) (ProbeRunPlan, string, error) {
	plan := heuristicProbePlan(in.UserText)
	enrichProbeFromSession(&plan, in.Session)
	applyProbeDefaultSignal(&plan)
	if heuristicProbeSlotsReady(plan) {
		return plan, fmt.Sprintf("playbookexec plan(heuristic): stock=%q signal=%q kind=%s",
			plan.StockQuery, plan.SignalQuery, plan.SignalKind), nil
	}
	if r == nil || r.Gateway == nil {
		if !slots.StockQueryPlausible(plan.StockQuery) {
			return plan, "", fmt.Errorf("缺少标的，请说明要测哪只股票的买卖点")
		}
		if strings.TrimSpace(plan.SignalQuery) == "" {
			return plan, "", fmt.Errorf("缺少信号，请说明要用哪种策略探测")
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
		{Role: llm.RoleSystem, Content: probePlanSystem},
		{Role: llm.RoleUser, Content: planUser},
	}
	callCtx := llm.WithCallMeta(ctx, llm.CallMeta{Kind: llm.TaskChat, ToolSchemaCount: 0})
	resp, err := r.Gateway.ChatStream(callCtx, messages, nil, in.Session.ID, step, nil)
	if err != nil {
		return plan, "", err
	}
	parsed, err := parseProbePlanJSON(resp.Content)
	if err != nil {
		if !slots.StockQueryPlausible(plan.StockQuery) {
			return plan, "", fmt.Errorf("计划解析失败：%w", err)
		}
		applyProbeDefaultSignal(&plan)
		if strings.TrimSpace(plan.SignalQuery) == "" {
			return plan, "", fmt.Errorf("缺少信号，请说明要用哪种策略探测")
		}
		return plan, "playbookexec plan(heuristic fallback)", nil
	}
	mergeProbePlan(&plan, parsed)
	applyProbeDefaultSignal(&plan)
	if !slots.StockQueryPlausible(plan.StockQuery) {
		return plan, "", fmt.Errorf("缺少标的，请说明要测哪只股票的买卖点")
	}
	if strings.TrimSpace(plan.SignalQuery) == "" {
		return plan, "", fmt.Errorf("缺少信号，请说明要用哪种策略探测")
	}
	return plan, fmt.Sprintf("playbookexec plan(llm): stock=%q signal=%q kind=%s",
		plan.StockQuery, plan.SignalQuery, plan.SignalKind), nil
}

func heuristicProbeSlotsReady(plan ProbeRunPlan) bool {
	return slots.StockQueryPlausible(plan.StockQuery) && strings.TrimSpace(plan.SignalQuery) != ""
}

func parseProbePlanJSON(raw string) (ProbeRunPlan, error) {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, "{"); i >= 0 {
		if j := strings.LastIndex(raw, "}"); j > i {
			raw = raw[i : j+1]
		}
	}
	var wire struct {
		StockQuery  string `json:"stock_query"`
		SignalQuery string `json:"signal_query"`
		SignalKind  string `json:"signal_kind"`
		MonthsBack  int    `json:"months_back"`
		Frequency   string `json:"frequency"`
	}
	if err := json.Unmarshal([]byte(raw), &wire); err != nil {
		return ProbeRunPlan{}, err
	}
	plan := ProbeRunPlan{
		StockQuery:  strings.TrimSpace(wire.StockQuery),
		SignalQuery: strings.TrimSpace(wire.SignalQuery),
		SignalKind:  strings.TrimSpace(wire.SignalKind),
		MonthsBack:  wire.MonthsBack,
		Frequency:   strings.TrimSpace(wire.Frequency),
	}
	normalizeProbePlan(&plan)
	return plan, nil
}

func mergeProbePlan(dst *ProbeRunPlan, src ProbeRunPlan) {
	if strings.TrimSpace(src.StockQuery) != "" {
		dst.StockQuery = strings.TrimSpace(src.StockQuery)
	}
	if strings.TrimSpace(src.SignalQuery) != "" {
		dst.SignalQuery = strings.TrimSpace(src.SignalQuery)
	}
	if strings.TrimSpace(src.SignalKind) != "" {
		dst.SignalKind = strings.TrimSpace(src.SignalKind)
	}
	if src.MonthsBack > 0 {
		dst.MonthsBack = src.MonthsBack
	}
	if strings.TrimSpace(src.Frequency) != "" {
		dst.Frequency = strings.TrimSpace(src.Frequency)
	}
	normalizeProbePlan(dst)
}

// applyProbeDefaultSignal fills the platform default combination when the user asks
// for buy/sell probe on a stock but names no strategy (matches trading_operation UI).
func applyProbeDefaultSignal(plan *ProbeRunPlan) {
	if plan == nil || strings.TrimSpace(plan.SignalQuery) != "" {
		return
	}
	plan.SignalQuery = defaultProbeCombinationQuery
	plan.SignalKind = "combination"
	normalizeProbePlan(plan)
}

func normalizeProbePlan(plan *ProbeRunPlan) {
	if plan.MonthsBack <= 0 {
		plan.MonthsBack = 3
	}
	if strings.TrimSpace(plan.SignalKind) == "" {
		upper := strings.ToUpper(plan.SignalQuery)
		if strings.Contains(upper, "SAR") && strings.Contains(upper, "MACD") {
			plan.SignalKind = "combination"
		} else {
			plan.SignalKind = "indicator"
		}
	}
}
