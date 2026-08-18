package premarket

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/stockfmt"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/textutil"
)

func extractKeyLevels(ws memory.StockWorkspace) stockfmt.KeyLevels {
	levels := stockfmt.ExtractWeeklyKeyLevels(ws.WeeklyAnalysisRef, ws.Code)
	return levels
}

func buildSubstantiveReason(ws memory.StockWorkspace, levels stockfmt.KeyLevels, marketResult string) string {
	attitude := ws.Attitude
	if attitude == "" {
		attitude = "neutral"
	}
	parts := []string{fmt.Sprintf("Bot 昨日态度为 %s", stockfmt.LocalizeAttitude(attitude))}

	if levels.Valid {
		parts = append(parts, fmt.Sprintf("周线关键支撑约 %.2f、阻力约 %.2f", *levels.Support, *levels.Resistance))
	} else if excerpt := weeklyConclusionExcerpt(ws.WeeklyAnalysisRef); excerpt != "" {
		parts = append(parts, excerpt)
	}

	capText := capitalInterpretationText(ws)
	if capText != "" {
		parts = append(parts, capText)
		if stockfmt.CapitalFlowDivergent(capText) {
			parts = append(parts, "资金流多空分歧明显")
		}
	}

	if market := strings.TrimSpace(marketResult); market != "" {
		parts = append(parts, fmt.Sprintf("市场盘前方向为 %s", localizeMarketResult(market)))
	}

	reason := strings.Join(parts, "；") + "。"
	if len([]rune(reason)) < 80 {
		reason += "建议结合盘中量价与大盘联动验证后再做操作决策。"
	}
	return reason
}

func buildStockSummaryOneLiner(ws memory.StockWorkspace, result, suggestion string) string {
	name := displayStockName(ws, ws.Code)
	dir := map[string]string{"long": "偏多", "short": "偏空", "neutral": "中性"}[result]
	sug := map[string]string{"buy": "可考虑买入", "sell": "可考虑减仓", "hold": "建议观望"}[suggestion]
	parts := []string{fmt.Sprintf("%s盘前研判%s", name, dir)}
	if sug != "" {
		parts = append(parts, sug)
	}
	capText := capitalInterpretationText(ws)
	if stockfmt.CapitalFlowDivergent(capText) {
		parts = append(parts, "资金流存在分歧")
	}
	return textutil.OneLine(strings.Join(parts, "，")+"。", 200)
}

func finalizeSuggestion(result, confidence, marketResult string, capitalDivergent bool) string {
	base := suggestionFor(result)
	if base == "hold" {
		return "hold"
	}
	conf := strings.ToLower(strings.TrimSpace(confidence))
	if conf == "review_required" || conf == "low" {
		return "hold"
	}
	if capitalDivergent {
		return "hold"
	}
	market := strings.ToLower(strings.TrimSpace(marketResult))
	if market == "neutral" && conf != "high" {
		return "hold"
	}
	return base
}

func capitalInterpretationText(ws memory.StockWorkspace) string {
	for _, raw := range []string{ws.CapitalDistributionSummary, ws.CapitalFlowSummary} {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		for _, line := range strings.Split(raw, "\n") {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "简要解读") {
				return textutil.OneLine(strings.TrimPrefix(line, "**简要解读**："), 120)
			}
		}
		return textutil.OneLine(raw, 120)
	}
	return ""
}

func weeklyConclusionExcerpt(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "结论") {
			line = strings.TrimLeft(line, "#-* ")
			return textutil.OneLine(line, 100)
		}
	}
	lines := strings.Split(text, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if len([]rune(line)) >= 20 {
			return textutil.OneLine(line, 100)
		}
	}
	return ""
}

func localizeMarketResult(result string) string {
	switch strings.ToLower(strings.TrimSpace(result)) {
	case "long", "bullish":
		return "偏多"
	case "short", "bearish":
		return "偏空"
	default:
		return "中性"
	}
}

func keyLevelWatchPoint(levels stockfmt.KeyLevels) string {
	if !levels.Valid {
		return "关注周线关键支撑/阻力是否被放量突破或跌破。"
	}
	return fmt.Sprintf("关注支撑位 %.2f 与阻力位 %.2f 是否被放量突破或跌破。", *levels.Support, *levels.Resistance)
}

func isBoilerplateReason(reason string) bool {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return true
	}
	return strings.Contains(reason, "证据已纳入") && strings.Contains(reason, "条证据引用")
}
