package playbookexec

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

var backtestThinkBlockRE = regexp.MustCompile(`(?is)<(?:redacted_thinking|think)>(.*?)</(?:redacted_thinking|think)>`)

const backtestReplySystem = `你是 GeeGoo 策略回测助手。根据 JSON 为普通投资者写 Markdown 回复（中文）。

结构必须如下：
## {股票名} {代码} · 策略回测

### 一句话结论
1 句概括赚亏与风险（含收益率、最大回撤）

### 怎么测的
- 策略名称（中文）
- K 线周期、回测区间、初始资金、每次买入股数
- 止盈止损（人话，如「止盈5%，SAR动态止损」）
- 买卖条件（人话，不要写 type/param/index 英文字段）

### 结果怎么样
用表格列：指标 | 数值
必含：收益率、最终资产、最大回撤、成交笔数

### 成交记录
表格：时间 | 操作 | 价格 | 数量
最多 12 行；超出注明总笔数

### 小结
1～2 句 actionable 观察

禁止：log_id、signal_id、strategy_ids、months_back、frequency、JSON、trade_config 字段名、英文 API 参数。
禁止编造数据。
文末：仅供参考，非投资建议。`

type backtestReplyInput struct {
	Code           string
	Name           string
	Market         string
	StrategyLabel  string
	StrategyKind   string
	Frequency      string
	Period         string
	MonthsBack     int
	Fund           float64
	BaseOrderSize  int
	BuySignal      []any
	SellSignal     []any
	SignalID       string
	TradeConfig    map[string]any
	RunSummary     map[string]any
	LogDetail      map[string]any
}

func defaultSmartTradeTradeConfig() map[string]any {
	return map[string]any{
		"tp": map[string]any{
			"tp_switch":                 true,
			"tp_mode":                   "fix",
			"fix_tp":                    5,
			"profit_trailing":           true,
			"profit_trailing_deviation": 1,
		},
		"sl": map[string]any{
			"sl_switch":        true,
			"sl_mode":          "dynamic",
			"sl_dynamic_index": "SAR",
		},
		"position_mode": "fixed",
	}
}

func (r *Router) formatBacktestReply(ctx context.Context, in Input, step int, bundle backtestReplyInput) string {
	if r != nil && r.Gateway != nil {
		payload, err := json.Marshal(buildBacktestLLMPayload(bundle))
		if err == nil {
			messages := []llm.Message{
				{Role: llm.RoleSystem, Content: backtestReplySystem},
				{Role: llm.RoleUser, Content: "回测数据 JSON：\n" + string(payload)},
			}
			callCtx := llm.WithCallMeta(ctx, llm.CallMeta{Kind: llm.TaskChat, ToolSchemaCount: 0})
			resp, err := r.Gateway.ChatStream(callCtx, messages, nil, in.Session.ID, step, nil)
			if err == nil {
				if text := strings.TrimSpace(stripReasoningTags(resp.Content)); text != "" {
					return text
				}
			}
		}
	}
	return formatBacktestReplyFallback(bundle)
}

func buildBacktestLLMPayload(bundle backtestReplyInput) map[string]any {
	out := map[string]any{
		"stock": map[string]any{
			"name": bundle.Name,
			"code": bundle.Code,
		},
		"strategy": map[string]any{
			"name":      bundle.StrategyLabel,
			"buy_rule":  humanSignalRules(bundle.BuySignal, nil, ""),
			"sell_rule": humanSignalRules(bundle.SellSignal, nil, ""),
		},
		"settings": map[string]any{
			"kline":           humanFrequency(bundle.Frequency),
			"time_range":      humanPeriod(bundle.Period, bundle.MonthsBack),
			"initial_cash":    humanFund(bundle.Fund),
			"order_size":      humanOrderSize(bundle.BaseOrderSize),
			"risk_management": humanTradeConfigSummary(bundle.TradeConfig, bundle.BaseOrderSize),
		},
		"results": buildHumanResults(bundle),
	}
	if bundle.LogDetail != nil {
		out["trades"] = normalizeTradesForPayload(extractTradesList(bundle.LogDetail), 20)
	}
	return out
}

func buildHumanResults(bundle backtestReplyInput) map[string]any {
	run := bundle.RunSummary
	if run == nil {
		run = map[string]any{}
	}
	var resMap map[string]any
	if r := nestedMap(bundle.LogDetail, "run"); r != nil {
		if nested, ok := r["result"].(map[string]any); ok {
			resMap = nested
		}
	}
	get := func(keys ...string) any {
		for _, k := range keys {
			if v, ok := run[k]; ok && fmt.Sprint(v) != "" && fmt.Sprint(v) != "<nil>" {
				return v
			}
			if resMap != nil {
				if v, ok := resMap[k]; ok && fmt.Sprint(v) != "" && fmt.Sprint(v) != "<nil>" {
					return v
				}
			}
		}
		return nil
	}
	return map[string]any{
		"profit_rate_pct":  formatPercent(get("profit_rate")),
		"final_value":      formatMoney(get("final_value")),
		"max_drawdown_pct": formatPercent(get("drawdown")),
		"trade_count":      get("trade_count"),
		"closed_trades":    get("closed_trades"),
	}
}

func extractTradesList(logDetail map[string]any) []any {
	if logDetail == nil {
		return nil
	}
	if trades, ok := logDetail["trades"].([]any); ok {
		return trades
	}
	if run, ok := logDetail["run"].(map[string]any); ok {
		if trades, ok := run["trades"].([]any); ok {
			return trades
		}
	}
	return nil
}

func normalizeTradesForPayload(trades []any, max int) []map[string]any {
	if len(trades) == 0 {
		return nil
	}
	if max <= 0 {
		max = 30
	}
	if len(trades) > max {
		trades = trades[:max]
	}
	out := make([]map[string]any, 0, len(trades))
	for _, raw := range trades {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, map[string]any{
			"time":   firstNonEmpty(row["time"], row["at"], row["datetime"]),
			"action": tradeSideLabel(row),
			"price":  firstNonEmpty(row["trade_price"], row["price"], row["close"]),
			"qty":    firstNonEmpty(row["position"], row["qty"], row["quantity"], row["size"]),
		})
	}
	return out
}

func nestedMap(v any, key string) map[string]any {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	child, ok := m[key].(map[string]any)
	if !ok {
		return nil
	}
	return child
}

func stripReasoningTags(text string) string {
	text = strings.TrimSpace(backtestThinkBlockRE.ReplaceAllString(text, ""))
	return strings.TrimSpace(text)
}

func formatBacktestReplyFallback(bundle backtestReplyInput) string {
	results := buildHumanResults(bundle)
	profit := fmt.Sprint(results["profit_rate_pct"])
	finalValue := fmt.Sprint(results["final_value"])
	drawdown := fmt.Sprint(results["max_drawdown_pct"])
	trades := fmt.Sprint(results["trade_count"])

	var b strings.Builder
	fmt.Fprintf(&b, "## %s %s · 策略回测\n\n", bundle.Name, bundle.Code)

	b.WriteString("### 一句话结论\n\n")
	fmt.Fprintf(&b, "在 %s、%s 条件下，收益率 **%s**，最大回撤 **%s**，共成交 **%s** 笔。\n\n",
		humanFrequency(bundle.Frequency), humanPeriod(bundle.Period, bundle.MonthsBack), profit, drawdown, trades)

	b.WriteString("### 怎么测的\n\n")
	fmt.Fprintf(&b, "- **策略**：%s\n", humanSignalRules(bundle.BuySignal, bundle.SellSignal, bundle.StrategyLabel))
	fmt.Fprintf(&b, "- **K 线 / 区间**：%s · %s\n", humanFrequency(bundle.Frequency), humanPeriod(bundle.Period, bundle.MonthsBack))
	fmt.Fprintf(&b, "- **资金**：初始 %s，%s\n", humanFund(bundle.Fund), humanOrderSize(bundle.BaseOrderSize))
	fmt.Fprintf(&b, "- **风控**：%s\n\n", humanTradeConfigSummary(bundle.TradeConfig, bundle.BaseOrderSize))

	b.WriteString("### 结果怎么样\n\n")
	b.WriteString("| 指标 | 数值 |\n| --- | --- |\n")
	fmt.Fprintf(&b, "| 收益率 | %s |\n", profit)
	fmt.Fprintf(&b, "| 最终资产 | %s |\n", finalValue)
	fmt.Fprintf(&b, "| 最大回撤 | %s |\n", drawdown)
	fmt.Fprintf(&b, "| 成交笔数 | %s |\n\n", trades)

	if table := formatTradesTable(bundle.LogDetail); table != "" {
		b.WriteString("### 成交记录\n\n")
		b.WriteString(strings.Replace(table, "| 方向 |", "| 操作 |", 1))
		b.WriteString("\n\n")
	}

	b.WriteString("### 小结\n\n")
	fmt.Fprintf(&b, "本次回测最终资产 %s，回撤 %s。", finalValue, drawdown)
	if bundle.SellSignal != nil {
		b.WriteString(" 以上为历史模拟结果，不代表未来表现。")
	}
	b.WriteString("\n\n> 仅供参考，非投资建议。")
	return b.String()
}

func formatTradesTable(logDetail map[string]any) string {
	trades := extractTradesList(logDetail)
	if len(trades) == 0 {
		return ""
	}
	limit := len(trades)
	suffix := ""
	if limit > 15 {
		limit = 15
		suffix = fmt.Sprintf("\n\n共 %d 笔，仅列前 15 笔。", len(trades))
	}
	var b strings.Builder
	b.WriteString("| 时间 | 操作 | 价格 | 数量 |\n| --- | --- | --- | --- |\n")
	for i := 0; i < limit; i++ {
		row, ok := trades[i].(map[string]any)
		if !ok {
			continue
		}
		fmt.Fprintf(&b, "| %v | %s | %v | %v |\n",
			firstNonEmpty(row["time"], row["at"], row["datetime"]),
			tradeSideLabel(row),
			firstNonEmpty(row["trade_price"], row["price"], row["close"]),
			firstNonEmpty(row["position"], row["qty"], row["quantity"], row["size"]),
		)
	}
	b.WriteString(suffix)
	return strings.TrimSpace(b.String())
}

func tradeSideLabel(row map[string]any) string {
	if label := strings.TrimSpace(fmt.Sprint(row["action_label"])); label != "" && label != "<nil>" {
		return label
	}
	action := strings.ToLower(strings.TrimSpace(fmt.Sprint(row["action"])))
	switch {
	case strings.Contains(action, "buy"), action == "signalbuy":
		return "买"
	case strings.Contains(action, "sell"), strings.Contains(action, "stop"), strings.Contains(action, "profit"), action == "stoploss", action == "takeprofit":
		return "卖"
	}
	for _, key := range []string{"side", "direction", "type"} {
		v := strings.ToLower(strings.TrimSpace(fmt.Sprint(row[key])))
		switch v {
		case "buy", "b", "1", "long":
			return "买"
		case "sell", "s", "-1", "short":
			return "卖"
		}
	}
	if sig, ok := row["signal"].(float64); ok {
		if sig > 0 {
			return "买"
		}
		if sig < 0 {
			return "卖"
		}
	}
	return "—"
}

func firstNonEmpty(values ...any) any {
	for _, v := range values {
		s := strings.TrimSpace(fmt.Sprint(v))
		if s != "" && s != "<nil>" {
			return v
		}
	}
	return ""
}

func signalIDFromRow(row map[string]any) string {
	if row == nil {
		return ""
	}
	for _, key := range []string{"signal_id", "id", "_id"} {
		if id := strings.TrimSpace(fmt.Sprint(row[key])); id != "" && id != "<nil>" {
			return id
		}
	}
	return ""
}
