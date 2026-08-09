package workflow

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/stockfmt"
)

// DecideIntraday applies geegoo intraday decision rules (Step 5.5).
func DecideIntraday(ws memory.StockWorkspace) (result, confidence string) {
	return memory.DecideIntraday(ws)
}

// SessionBiasFromChangePct maps change_pct to session_bias per geegoo post-market rules.
func SessionBiasFromChangePct(pct float64) string {
	if pct > 1 {
		return "bullish"
	}
	if pct < -1 {
		return "bearish"
	}
	return "neutral"
}

// VsPreMarket compares premarket_market result with session_bias.
func VsPreMarket(preResult, sessionBias string) string {
	if preResult == "" {
		return "na"
	}
	pre := strings.ToLower(strings.TrimSpace(preResult))
	bias := strings.ToLower(strings.TrimSpace(sessionBias))
	aligned := (pre == "long" && bias == "bullish") ||
		(pre == "short" && bias == "bearish") ||
		(pre == "neutral" && bias == "neutral")
	if aligned {
		return "aligned"
	}
	contradicted := (pre == "long" && bias == "bearish") || (pre == "short" && bias == "bullish")
	if contradicted {
		return "contradicted"
	}
	return "partial"
}

// BotLogType maps bot_type to getBotLogByType type parameter.
func BotLogType(botType string) string {
	bt := strings.ToUpper(botType)
	if strings.Contains(bt, "GRID") {
		return "GRID"
	}
	return "DCA"
}

// MarketSummaryFromHourly builds a short human-readable market recap (not raw MCP dump).
func MarketSummaryFromHourly(ws memory.StockWorkspace) string {
	bias := localizeSessionBias(ws.SessionBias)
	var parts []string
	if ws.ChangePct != 0 {
		word := "收涨"
		if ws.ChangePct < 0 {
			word = "收跌"
		}
		parts = append(parts, fmt.Sprintf("今日%s %.2f%%", word, abs(ws.ChangePct)))
	}
	if bias != "" {
		parts = append(parts, fmt.Sprintf("盘面倾向%s", bias))
	}
	if line := stockfmt.ExtractTaggedConclusion(ws.HourlySignalAnalysis); line != "" {
		parts = append(parts, line)
	} else if line := stockfmt.ExtractTaggedConclusion(ws.HourlyPriceAnalysis); line != "" {
		parts = append(parts, line)
	} else if line := stockfmt.FirstSentence(stockfmt.PostmarketSummaryExcerpt(ws.HourlyPriceAnalysis, 280)); line != "" {
		parts = append(parts, line)
	}
	if len(parts) == 0 {
		return "今日行情数据不完整，建议结合正文小时级分析综合判断。"
	}
	text := strings.Join(parts, "，")
	if !strings.HasSuffix(text, "。") {
		text += "。"
	}
	if len([]rune(text)) < 80 {
		if ex := stockfmt.PostmarketSummaryExcerpt(ws.HourlyPriceAnalysis, 200); ex != "" {
			text += stockfmt.FirstSentence(ex)
			if !strings.HasSuffix(text, "。") {
				text += "。"
			}
		}
	}
	for len([]rune(text)) < 80 {
		text += "短线波动与量能变化需结合关键位继续观察。"
	}
	return text
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// TradeSummaryFromBotLog builds trade_summary from bot log snapshot.
func TradeSummaryFromBotLog(ws memory.StockWorkspace) string {
	summary := strings.TrimSpace(ws.BotLogSummary)
	if summary == "" || summary == "[]" || summary == "{}" || summary == "map[]" || len([]rune(summary)) < 40 {
		if ws.HasPosition {
			return fmt.Sprintf("当前持仓：%s", ws.PositionSummary)
		}
		return "当日机器人未产生成交记录，持仓与策略状态保持不变，可关注下一交易日信号触发与仓位变化。"
	}
	return oneLine(summary, 400)
}

// ExperienceSummaryDefault builds a post-market experience paragraph.
func ExperienceSummaryDefault(ws memory.StockWorkspace, vs string) string {
	bias := stockfmt.LocalizeAttitude(ws.SessionBias)
	if bias == "" || bias == ws.SessionBias {
		bias = localizeSessionBias(ws.SessionBias)
	}
	return fmt.Sprintf(
		"今日盘面倾向为%s，与盘前对照结论为%s。复盘时应优先核对盘前观点与盘中实际走势是否一致，"+
			"并记录 Bot（%s）在当日信号触发与执行上的偏差，便于后续调整策略开关或止盈止损参数。",
		bias, localizeVsPreMarket(vs), strings.TrimSpace(ws.BotType),
	)
}

func localizeSessionBias(bias string) string {
	switch strings.ToLower(strings.TrimSpace(bias)) {
	case "bullish":
		return "偏多"
	case "bearish":
		return "偏空"
	case "neutral", "":
		return "中性"
	default:
		return bias
	}
}

func localizeVsPreMarket(vs string) string {
	switch strings.ToLower(strings.TrimSpace(vs)) {
	case "aligned":
		return "一致"
	case "partial":
		return "部分一致"
	case "contradicted":
		return "相矛盾"
	case "na", "":
		return "无法对照"
	default:
		return vs
	}
}
