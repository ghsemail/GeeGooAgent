package playbookexec

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// BacktestRunPlan is the structured plan extracted before deterministic execution.
type BacktestRunPlan struct {
	StockQuery   string  `json:"stock_query"`
	SignalQuery  string  `json:"signal_query"`
	SignalKind   string  `json:"signal_kind"` // combination | indicator
	Period       string  `json:"period"`
	MonthsBack   int     `json:"months_back"`
	Fund         float64 `json:"fund"`
}

const backtestPlanSystem = `你是 GeeGoo 策略回测计划解析器。只输出一个 JSON 对象，不要 markdown，不要解释。
字段：
- stock_query: 用户提到的标的名称或代码片段（如 小米、AAPL、00700）
- signal_query: 信号描述（如 SAR+MACD、RSI阈值）
- signal_kind: combination 或 indicator
- period: 1m/2m/3m，默认 3m
- months_back: 整数，默认 3
- fund: 初始资金，默认 100000
用户未提到的字段用默认值。`

var (
	reMonths         = regexp.MustCompile(`(?i)(\d+)\s*(个)?月`)
	reFund           = regexp.MustCompile(`(?i)(?:资金|本金|fund)?\s*(\d+)\s*(?:万|w)?`)
	reStrategyQuoted = regexp.MustCompile(`「([^」]+)」`)
	reParenTicker    = regexp.MustCompile(`[（(]\s*([A-Z]{1,5})(?:\s*[·\.]\s*[^）)]*)?[）)]`)
	reTicker         = regexp.MustCompile(`\b([A-Z]{1,5})(?:\.US)?\b`)
	reHKCode         = regexp.MustCompile(`(\d{4,5})(?:\.HK)?`)
)

var stockQueryBlocklist = map[string]struct{}{
	"MACD": {}, "SAR": {}, "RSI": {}, "KDJ": {}, "EMA": {}, "SMA": {},
	"BOLL": {}, "DMI": {}, "ATR": {}, "OBV": {}, "US": {}, "HK": {},
}

func (r *Router) buildBacktestPlan(ctx context.Context, in Input, step int) (BacktestRunPlan, string, error) {
	plan := heuristicBacktestPlan(in.UserText)
	enrichPlanFromSession(&plan, in.Session)
	if strings.TrimSpace(plan.StockQuery) != "" && strings.TrimSpace(plan.SignalQuery) != "" {
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
	upper := strings.ToUpper(msg)

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

	plan.StockQuery = extractStockQuery(msg)
	if m := reStrategyQuoted.FindStringSubmatch(msg); len(m) > 1 {
		if name := strings.TrimSpace(m[1]); name != "" {
			plan.SignalKind = "combination"
			plan.SignalQuery = name
		}
	}
	if strings.TrimSpace(plan.SignalQuery) == "" && strings.Contains(upper, "SAR") && strings.Contains(upper, "MACD") {
		plan.SignalKind = "combination"
		plan.SignalQuery = "SAR MACD"
	} else if strings.TrimSpace(plan.SignalQuery) == "" && strings.Contains(upper, "RSI") {
		plan.SignalKind = "indicator"
		plan.SignalQuery = "RSI"
	} else if strings.TrimSpace(plan.SignalQuery) == "" && strings.Contains(upper, "SAR") {
		plan.SignalKind = "indicator"
		plan.SignalQuery = "SAR"
	} else if strings.TrimSpace(plan.SignalQuery) == "" && (strings.Contains(msg, "组合") || strings.Contains(msg, "共振")) {
		plan.SignalKind = "combination"
		plan.SignalQuery = msg
	}
	return plan
}

func extractStockQuery(msg string) string {
	if m := reParenTicker.FindStringSubmatch(msg); len(m) > 1 {
		if ticker := strings.ToUpper(strings.TrimSpace(m[1])); !isBlockedStockToken(ticker) {
			return ticker
		}
	}
	for _, key := range []string{"小米", "腾讯", "茅台", "苹果", "特斯拉", "英伟达", "微软", "谷歌", "阿里", "拼多多"} {
		if strings.Contains(msg, key) {
			return key
		}
	}
	if regexp.MustCompile(`(?i)\bApple\b`).MatchString(msg) {
		return "苹果"
	}
	upper := strings.ToUpper(msg)
	for _, m := range reTicker.FindAllStringSubmatch(upper, -1) {
		if len(m) > 1 && !isBlockedStockToken(m[1]) {
			return m[1]
		}
	}
	if m := reHKCode.FindStringSubmatch(msg); len(m) > 1 {
		return m[1]
	}
	return ""
}

func isBlockedStockToken(token string) bool {
	_, ok := stockQueryBlocklist[strings.ToUpper(strings.TrimSpace(token))]
	return ok
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
