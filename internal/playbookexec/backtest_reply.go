package playbookexec

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

const backtestReplySystem = `你是 GeeGoo 策略回测助手。根据提供的 JSON 回测数据写 Markdown 回复（中文），必须包含：
1. 标题：标的名称/代码 + 策略名
2. **回测配置**：frequency、months_back/period、fund、base_order_size、trade_config 要点、买卖规则概要（index 链）
3. **结果摘要**：收益率、最终资产、最大回撤、成交笔数、log_id
4. **成交明细表**（若有 trades）：列 时间 | 方向(买/卖) | 价格 | 数量；最多展示 15 笔，超出说明「共 N 笔，仅列前 15 笔」
5. 1～2 句简短结论
文末加「仅供参考，非投资建议。」
禁止编造 JSON 中不存在的数据；数值与 log_id 必须与 JSON 一致。`

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
		payload, err := json.Marshal(bundle)
		if err == nil {
			messages := []llm.Message{
				{Role: llm.RoleSystem, Content: backtestReplySystem},
				{Role: llm.RoleUser, Content: "回测数据 JSON：\n" + truncate(string(payload), 12000)},
			}
			callCtx := llm.WithCallMeta(ctx, llm.CallMeta{Kind: llm.TaskChat, ToolSchemaCount: 0})
			resp, err := r.Gateway.ChatStream(callCtx, messages, nil, in.Session.ID, step, nil)
			if err == nil {
				if text := strings.TrimSpace(resp.Content); text != "" {
					return text
				}
			}
		}
	}
	return formatBacktestReplyFallback(bundle)
}

func formatBacktestReplyFallback(bundle backtestReplyInput) string {
	run := bundle.RunSummary
	if run == nil {
		run = map[string]any{}
	}
	profit := fmt.Sprint(firstNonEmpty(run["profit_rate"], nestedAny(run, "result", "profit_rate")))
	finalValue := fmt.Sprint(firstNonEmpty(run["final_value"], nestedAny(run, "result", "final_value")))
	drawdown := fmt.Sprint(firstNonEmpty(run["drawdown"], nestedAny(run, "result", "drawdown")))
	trades := fmt.Sprint(firstNonEmpty(run["trade_count"], nestedAny(run, "result", "trade_count")))
	logID := fmt.Sprint(run["log_id"])

	var b strings.Builder
	fmt.Fprintf(&b, "## %s %s · SmartTrade 回测\n\n", bundle.Name, bundle.Code)
	fmt.Fprintf(&b, "- **策略**：%s\n", bundle.StrategyLabel)
	fmt.Fprintf(&b, "- **周期**：%s · 回溯 %s（months_back=%d）\n", bundle.Frequency, bundle.Period, bundle.MonthsBack)
	fmt.Fprintf(&b, "- **资金**：%.0f · 每次买入 %d 股\n", bundle.Fund, bundle.BaseOrderSize)
	if bundle.SignalID != "" {
		fmt.Fprintf(&b, "- **关联 signal_id**：%s\n", bundle.SignalID)
	}
	fmt.Fprintf(&b, "- **收益率**：%s%%\n", profit)
	fmt.Fprintf(&b, "- **最终资产**：%s\n", finalValue)
	fmt.Fprintf(&b, "- **最大回撤**：%s%%\n", drawdown)
	fmt.Fprintf(&b, "- **成交笔数**：%s\n", trades)
	fmt.Fprintf(&b, "- **log_id**：%s\n\n", logID)

	if tc := summarizeTradeConfig(bundle.TradeConfig); tc != "" {
		b.WriteString("### 交易参数\n\n")
		b.WriteString(tc)
		b.WriteString("\n\n")
	}
	if rules := summarizeSignalRules(bundle.BuySignal, bundle.SellSignal); rules != "" {
		b.WriteString("### 买卖规则\n\n")
		b.WriteString(rules)
		b.WriteString("\n\n")
	}
	if table := formatTradesTable(bundle.LogDetail); table != "" {
		b.WriteString("### 成交明细\n\n")
		b.WriteString(table)
		b.WriteString("\n\n")
	}
	b.WriteString("> 仅供参考，非投资建议。")
	return b.String()
}

func summarizeTradeConfig(tc map[string]any) string {
	if len(tc) == 0 {
		return ""
	}
	raw, err := json.Marshal(tc)
	if err != nil {
		return ""
	}
	return "```json\n" + string(raw) + "\n```"
}

func summarizeSignalRules(buy, sell []any) string {
	desc := func(label string, rules []any) string {
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
			typ := strings.TrimSpace(fmt.Sprint(row["type"]))
			if idx == "" {
				continue
			}
			parts = append(parts, fmt.Sprintf("%s(%s)", idx, typ))
		}
		if len(parts) == 0 {
			return ""
		}
		return label + ": " + strings.Join(parts, " → ")
	}
	buyLine := desc("买", buy)
	sellLine := desc("卖", sell)
	if buyLine == "" && sellLine == "" {
		return ""
	}
	if sellLine == "" || sellLine == buyLine {
		return buyLine
	}
	return buyLine + "\n" + sellLine
}

func formatTradesTable(logDetail map[string]any) string {
	if logDetail == nil {
		return ""
	}
	raw, ok := logDetail["trades"].([]any)
	if !ok || len(raw) == 0 {
		return ""
	}
	limit := len(raw)
	suffix := ""
	if limit > 15 {
		limit = 15
		suffix = fmt.Sprintf("\n\n共 %d 笔，仅列前 15 笔。", len(raw))
	}
	var b strings.Builder
	b.WriteString("| 时间 | 方向 | 价格 | 数量 |\n| --- | --- | --- | --- |\n")
	for i := 0; i < limit; i++ {
		row, ok := raw[i].(map[string]any)
		if !ok {
			continue
		}
		side := tradeSideLabel(row)
		fmt.Fprintf(&b, "| %v | %s | %v | %v |\n",
			firstNonEmpty(row["time"], row["at"], row["datetime"]),
			side,
			firstNonEmpty(row["price"], row["close"]),
			firstNonEmpty(row["qty"], row["quantity"], row["size"]),
		)
	}
	b.WriteString(suffix)
	return strings.TrimSpace(b.String())
}

func tradeSideLabel(row map[string]any) string {
	for _, key := range []string{"side", "direction", "action", "type"} {
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

func nestedAny(v any, keys ...string) any {
	cur := v
	for _, key := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[key]
	}
	return cur
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
