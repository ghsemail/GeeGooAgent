package stockfmt

import (
	"regexp"
	"strings"
)

var (
	intradayTableRowRE   = regexp.MustCompile(`^\|?.+\|.+\|`)
	intradayMetricRowRE  = regexp.MustCompile(`(?i)(EMA|MACD|KDJ|RSI|SAR|VWAP|布林带|吊灯|平均K线|指标)`)
	intradaySectionHeadRE = regexp.MustCompile(`(?i)(综合|总结|判定|展望|结论|风险提示|走势概览|技术面)`)
)

// FormatIntradayHourlySection condenses hourly MCP price/kline text to prose (no tables).
func FormatIntradayHourlySection(raw, fallback string) string {
	return formatIntradaySection(raw, fallback, false)
}

// FormatIntradaySignalSection condenses hourly signal MCP output to a short verdict paragraph.
func FormatIntradaySignalSection(raw string) string {
	return formatIntradaySection(raw, "暂无小时级信号分析。", true)
}

func formatIntradaySection(raw, fallback string, signalOnly bool) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	if c := ExtractTaggedConclusion(raw); c != "" {
		return PolishStockPremarketMarkdown(c)
	}
	parts := extractIntradayNarrativeBlocks(raw, signalOnly)
	if len(parts) == 0 {
		if ex := PostmarketSummaryExcerpt(FormatEmbeddedHourlyAnalysis(raw), 520); ex != "" {
			return PolishStockPremarketMarkdown(ex)
		}
		return fallback
	}
	text := strings.Join(parts, "\n\n")
	return PolishStockPremarketMarkdown(text)
}

func extractIntradayNarrativeBlocks(raw string, signalOnly bool) []string {
	lines := strings.Split(raw, "\n")
	var blocks []string
	var buf []string
	flush := func() {
		if len(buf) == 0 {
			return
		}
		p := strings.TrimSpace(strings.Join(buf, " "))
		p = strings.Trim(p, "*# ")
		if p != "" && !isIntradayNoiseLine(p) {
			blocks = append(blocks, p)
		}
		buf = buf[:0]
	}
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || trim == "---" {
			flush()
			continue
		}
		if strings.HasPrefix(trim, "#") {
			flush()
			title := strings.TrimLeft(trim, "# ")
			if intradaySectionHeadRE.MatchString(title) || (!signalOnly && strings.Contains(title, "走势")) {
				buf = append(buf, title+"：")
			}
			continue
		}
		if isIntradayNoiseLine(trim) {
			continue
		}
		if signalOnly && intradayMetricRowRE.MatchString(trim) && !intradaySectionHeadRE.MatchString(trim) {
			continue
		}
		trim = strings.TrimPrefix(strings.TrimPrefix(trim, "- "), "• ")
		trim = strings.Trim(trim, "* ")
		if trim == "" {
			continue
		}
		buf = append(buf, trim)
	}
	flush()
	if len(blocks) > 3 {
		blocks = blocks[:3]
	}
	return blocks
}

func isIntradayNoiseLine(line string) bool {
	trim := strings.TrimSpace(line)
	if trim == "" {
		return true
	}
	if intradayTableRowRE.MatchString(trim) {
		return true
	}
	if isGarbageTableLine(trim) {
		return true
	}
	if strings.HasPrefix(trim, "|") && strings.Contains(trim, "|") {
		return true
	}
	if signalMetricLineRE.MatchString(trim) {
		return true
	}
	if strings.Count(trim, "：") >= 3 && strings.Count(trim, "|") >= 1 {
		return true
	}
	if looksTruncatedFragment(strings.Trim(trim, "*# ")) {
		return true
	}
	if strings.Contains(trim, "|---") || strings.Contains(trim, "------") {
		return true
	}
	return false
}

// LocalizeDecisionTerms replaces English decision enums in user-facing prose.
func LocalizeDecisionTerms(text string) string {
	repl := []struct{ old, neu string }{
		{"high", "高"}, {"medium", "中"}, {"low", "低"},
		{"buy", "买入"}, {"sell", "卖出"}, {"hold", "观望"},
		{"long", "看多"}, {"short", "看空"}, {"neutral", "中性"},
		{"bullish", "偏多"}, {"bearish", "偏空"},
	}
	out := text
	for _, r := range repl {
		out = regexp.MustCompile(`(?i)\b`+regexp.QuoteMeta(r.old)+`\b`).ReplaceAllString(out, r.neu)
	}
	return PolishStockPremarketMarkdown(out)
}

// IntradayAPISummary builds a short plain-text summary without markdown or tags.
func IntradayAPISummary(stockName, code, tradeType, result, confidence, reportBody string) string {
	parts := []string{
		strings.TrimSpace(stockName) + "（" + strings.TrimSpace(code) + "）",
		"信号" + strings.TrimSpace(tradeType),
		"决策" + localizeDecisionWord(result) + "，置信度" + localizeDecisionWord(confidence),
	}
	if ex := PostmarketSummaryExcerpt(reportBody, 140); ex != "" {
		parts = append(parts, ex)
	}
	text := LocalizeDecisionTerms(strings.Join(parts, "。"))
	if !strings.HasSuffix(text, "。") {
		text += "。"
	}
	runes := []rune(text)
	if len(runes) > 200 {
		text = string(runes[:197]) + "…"
	}
	return text
}

func localizeDecisionWord(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "buy":
		return "买入"
	case "sell":
		return "卖出"
	case "hold":
		return "观望"
	case "high":
		return "高"
	case "medium":
		return "中"
	case "low":
		return "低"
	case "long":
		return "看多"
	case "short":
		return "看空"
	default:
		return strings.TrimSpace(v)
	}
}
