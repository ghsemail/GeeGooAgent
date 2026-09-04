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
	reMonths    = regexp.MustCompile(`(?i)(\d+)\s*(个)?月`)
	reFund      = regexp.MustCompile(`(?i)(?:资金|本金|fund)?\s*(\d+)\s*(?:万|w)?`)
	reUSTicker  = regexp.MustCompile(`\b([A-Z]{1,5})(?:\.US)?\b`)
	reHKCode    = regexp.MustCompile(`\b(\d{4,5})(?:\.HK)?\b`)
	reAShare    = regexp.MustCompile(`(?i)\b(\d{6})(?:\.(?:SZ|SH|BJ))?\b`)
	reCJKRun    = regexp.MustCompile(`\p{Han}{2,8}`)
	reIntentPad = regexp.MustCompile(`帮我回测一下|帮我测试一下|帮我回测|回测一下|跑回测|来回测|再回测|测试一下|看一下|就用刚才那套|刚才那套|用现成的来回测|不要新建`)
)

var tickerStopwords = map[string]struct{}{
	"DAILY": {}, "WEEKLY": {}, "MONTHLY": {}, "YEARLY": {},
	"MACD": {}, "RSI": {}, "SAR": {}, "EMA": {}, "SMA": {}, "KDJ": {}, "BOLL": {}, "ATR": {},
	"US": {}, "HK": {}, "CN": {}, "SZ": {}, "SH": {}, "BJ": {},
	"JSON": {}, "HTTP": {}, "HTML": {}, "SMART": {}, "TRADE": {},
	"AND": {}, "OR": {}, "THE": {}, "FOR": {}, "WITH": {},
}

var cjkStockStops = []string{
	"组合信号", "组合", "信号", "策略", "回测", "收益", "收益率", "回撤", "成交",
	"趋势", "直方图", "金叉", "死叉", "抛物线", "共振", "哪些", "现在", "标的",
	"股票", "一下", "请问", "帮我", "测试", "频率", "支持", "配套", "规则",
	"买入", "卖出", "简介", "当前", "全部", "三种", "共有",
}

var knownStockAliases = []string{
	"小米", "腾讯", "茅台", "苹果", "特斯拉", "英伟达", "微软", "谷歌", "阿里", "拼多多",
}

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

	plan.StockQuery = extractStockQuery(msg)
	applySignalHeuristics(&plan, msg)
	return plan
}

func extractStockQuery(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}
	for _, key := range knownStockAliases {
		if strings.Contains(msg, key) {
			return key
		}
	}
	if m := reAShare.FindStringSubmatch(msg); len(m) > 1 {
		if !(strings.Contains(msg, "资金") || strings.Contains(msg, "本金") || strings.Contains(strings.ToLower(msg), "fund")) {
			return strings.ToUpper(m[1])
		}
	}
	if m := reHKCode.FindStringSubmatch(msg); len(m) > 1 {
		return m[1]
	}
	upper := strings.ToUpper(msg)
	if m := reUSTicker.FindStringSubmatch(upper); len(m) > 1 {
		tok := m[1]
		if _, stop := tickerStopwords[tok]; !stop {
			return tok
		}
	}
	stripped := reIntentPad.ReplaceAllString(msg, " ")
	cjk := reCJKRun.FindAllString(stripped, -1)
	for i := len(cjk) - 1; i >= 0; i-- {
		if !isCJKStockStop(cjk[i]) {
			return cjk[i]
		}
	}
	return ""
}

func isCJKStockStop(s string) bool {
	for _, tok := range cjkStockStops {
		if s == tok || strings.Contains(s, tok) {
			return true
		}
	}
	return false
}

func looksLikeStockQuery(q string) bool {
	q = strings.TrimSpace(q)
	if q == "" {
		return false
	}
	if _, stop := tickerStopwords[strings.ToUpper(q)]; stop {
		return false
	}
	if isCJKStockStop(q) {
		return false
	}
	return true
}

func heuristicSlotsReady(plan BacktestRunPlan) bool {
	return looksLikeStockQuery(plan.StockQuery) && strings.TrimSpace(plan.SignalQuery) != ""
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
