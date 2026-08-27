package playbookexec

import (
	"fmt"
	"strings"
)

func humanFrequency(freq string) string {
	f := strings.TrimSpace(strings.ToLower(freq))
	switch f {
	case "5m":
		return "5分钟K线"
	case "15m":
		return "15分钟K线"
	case "60m":
		return "60分钟K线"
	case "daily", "1d", "d":
		return "日K"
	default:
		if f == "" {
			return "默认K线"
		}
		return freq + " K线"
	}
}

func humanMonthsRange(months int) string {
	if months <= 0 {
		months = 3
	}
	switch months {
	case 1:
		return "近1个月"
	case 3:
		return "近3个月"
	case 6:
		return "近6个月"
	case 12:
		return "近1年"
	default:
		return fmt.Sprintf("近%d个月", months)
	}
}

func humanPeriod(period string, monthsBack int) string {
	p := strings.TrimSpace(strings.ToLower(period))
	switch p {
	case "1m":
		return "近1个月"
	case "2m":
		return "近2个月"
	case "3m":
		return "近3个月"
	case "2w":
		return "近2周"
	case "":
		return humanMonthsRange(monthsBack)
	default:
		return humanMonthsRange(monthsBack)
	}
}

func humanFund(fund float64) string {
	if fund <= 0 {
		return "10万元"
	}
	if fund >= 10000 && int64(fund)%10000 == 0 {
		return fmt.Sprintf("%.0f万元", fund/10000)
	}
	return fmt.Sprintf("%.0f元", fund)
}

func humanOrderSize(size int) string {
	if size <= 0 {
		size = 100
	}
	return fmt.Sprintf("每次买入 %d 股", size)
}

func humanSignalRules(buy, sell []any, strategyLabel string) string {
	label := strings.TrimSpace(strategyLabel)
	if label != "" {
		return label
	}
	buyDesc := ruleChainLabel(buy)
	sellDesc := ruleChainLabel(sell)
	if sellDesc == "" || sellDesc == buyDesc {
		if buyDesc != "" {
			return buyDesc
		}
		return "自定义买卖规则"
	}
	return buyDesc + "（买） / " + sellDesc + "（卖）"
}

func ruleChainLabel(rules []any) string {
	parts := make([]string, 0, len(rules))
	for _, raw := range rules {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		idx := strings.TrimSpace(fmt.Sprint(row["index"]))
		if idx == "" || strings.EqualFold(idx, "nosignal") {
			continue
		}
		typ := strings.ToLower(strings.TrimSpace(fmt.Sprint(row["type"])))
		switch typ {
		case "flag":
			parts = append(parts, indexDisplayName(idx)+" 趋势")
		case "signal", "":
			parts = append(parts, indexDisplayName(idx))
		default:
			parts = append(parts, indexDisplayName(idx))
		}
	}
	return strings.Join(parts, " + ")
}

func indexDisplayName(index string) string {
	switch strings.ToUpper(strings.TrimSpace(index)) {
	case "MACD":
		return "MACD"
	case "SAR":
		return "SAR"
	case "RSI":
		return "RSI"
	case "RSICROSS":
		return "RSI 金叉死叉"
	default:
		return strings.TrimSpace(index)
	}
}

func humanTradeConfigSummary(tc map[string]any, orderSize int) string {
	if len(tc) == 0 {
		return humanOrderSize(orderSize)
	}
	var parts []string
	if tp, ok := tc["tp"].(map[string]any); ok {
		if mode := strings.TrimSpace(fmt.Sprint(tp["tp_mode"])); strings.EqualFold(mode, "fix") {
			if v := tp["fix_tp"]; v != nil {
				parts = append(parts, fmt.Sprintf("止盈 %v%%", v))
			}
		} else if mode != "" {
			parts = append(parts, "动态止盈")
		}
		if trailing, ok := tp["profit_trailing"].(bool); ok && trailing {
			parts = append(parts, "盈利回撤跟踪")
		}
	}
	if sl, ok := tc["sl"].(map[string]any); ok {
		if mode := strings.TrimSpace(fmt.Sprint(sl["sl_mode"])); strings.EqualFold(mode, "dynamic") {
			idx := strings.TrimSpace(fmt.Sprint(sl["sl_dynamic_index"]))
			if idx == "" {
				idx = "SAR"
			}
			parts = append(parts, fmt.Sprintf("%s 动态止损", indexDisplayName(idx)))
		} else if fix := sl["fix_sl"]; fix != nil {
			parts = append(parts, fmt.Sprintf("止损 %v%%", fix))
		}
	}
	parts = append(parts, humanOrderSize(orderSize))
	return strings.Join(parts, "；")
}

func formatPercent(v any) string {
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "" || s == "<nil>" {
		return "—"
	}
	if strings.Contains(s, "%") {
		return s
	}
	return s + "%"
}

func formatMoney(v any) string {
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "" || s == "<nil>" {
		return "—"
	}
	return s
}
