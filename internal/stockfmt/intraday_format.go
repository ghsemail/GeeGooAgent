package stockfmt

import (
	"regexp"
	"strings"
)

var (
	intradayTableRowRE      = regexp.MustCompile(`^\|?.+\|.+\|`)
	intradayMetricRowRE     = regexp.MustCompile(`(?i)(EMA|MACD|KDJ|RSI|SAR|VWAP|布林带|吊灯|平均K线|指标)`)
	intradaySectionHeadRE   = regexp.MustCompile(`(?i)(综合|总结|判定|展望|结论|风险提示|走势概览|技术面)`)
	intradayRawOHLCHeaderRE = regexp.MustCompile(`(?i)^(交易日|日期)[：:].*(开盘|收盘)`)
	intradayRawOHLCRowRE    = regexp.MustCompile(`^(\d{1,2}[/-]\d{1,2}|\d{2}-\d{2})[：:][\d.+%％\-]+[：:]`)
	intradayRawStatsHeaderRE = regexp.MustCompile(`(?i)^(信号类型|特征)[：:].*(数量|占比|表现|含义)`)
	intradayKlineDetailRowRE = regexp.MustCompile(`^\d{1,2}/\d{1,2}[：:].*(看多|看空|观望|对抗)`)
)

// FormatIntradayHourlySection condenses hourly MCP price/kline text to prose (no tables).
func FormatIntradayHourlySection(raw, fallback string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	if embedded := formatIntradayHourlyContent(raw); embedded != "" {
		return PolishStockPremarketMarkdown(embedded)
	}
	return formatIntradaySection(raw, fallback, false)
}

// FormatIntradaySignalSection condenses hourly signal MCP output to a short verdict paragraph.
func FormatIntradaySignalSection(raw string) string {
	if verdict := formatIntradayVerdictSection(raw); verdict != "" {
		return PolishStockPremarketMarkdown(verdict)
	}
	return formatIntradaySection(raw, "暂无小时级信号分析。", true)
}

func formatIntradayHourlyContent(raw string) string {
	embedded := FormatEmbeddedHourlyAnalysis(raw)
	if embedded == "" || embedded == "暂无周线技术分析。" {
		return ""
	}
	lines := strings.Split(embedded, "\n")
	out := make([]string, 0, len(lines))
	inCodeBlock := false
	skipDetailSection := false
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}
		if trim == "" {
			if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
				out = append(out, "")
			}
			continue
		}
		title := strings.Trim(trim, "*# ")
		if isIntradayDetailSectionTitle(title) {
			skipDetailSection = true
			continue
		}
		if isIntradayNarrativeSectionTitle(title) {
			skipDetailSection = false
		}
		if skipDetailSection {
			continue
		}
		if isIntradayRawDataLine(trim) {
			continue
		}
		out = append(out, trim)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func formatIntradayVerdictSection(raw string) string {
	lines := strings.Split(raw, "\n")
	var out []string
	inVerdict := false
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		if strings.HasPrefix(trim, "```") {
			continue
		}
		plain := strings.Trim(strings.TrimPrefix(trim, ">"), "*# ")
		if isIntradayVerdictHeading(plain) {
			inVerdict = true
			out = append(out, "**"+strings.TrimSuffix(strings.Trim(plain, "# "), "：")+"**")
			continue
		}
		if !inVerdict {
			continue
		}
		if isIntradayDetailSectionTitle(plain) && !isIntradayVerdictHeading(plain) {
			break
		}
		if polished := polishIntradayVerdictLine(trim); polished != "" {
			out = append(out, polished)
		}
	}
	if len(out) == 0 {
		if c := ExtractTaggedConclusion(raw); c != "" {
			return "**综合判定**\n\n" + polishIntradayVerdictLine(c)
		}
		return ""
	}
	return strings.Join(out, "\n\n")
}

func isIntradayVerdictHeading(title string) bool {
	title = strings.TrimSpace(strings.Trim(title, "#* "))
	return strings.Contains(title, "综合判定结论") ||
		strings.Contains(title, "综合多空判定") ||
		strings.Contains(title, "综合研判") ||
		strings.Contains(title, "综合判定")
}

func isIntradayDetailSectionTitle(title string) bool {
	title = strings.TrimSpace(strings.Trim(title, "#* "))
	switch {
	case strings.Contains(title, "日线级别分析"),
		strings.Contains(title, "分时点详情"),
		strings.Contains(title, "信号统计") && strings.Contains(title, "一、"),
		strings.Contains(title, "各技术指标最新状态"):
		return true
	default:
		return false
	}
}

func isIntradayNarrativeSectionTitle(title string) bool {
	title = strings.TrimSpace(strings.Trim(title, "#* "))
	for _, kw := range []string{
		"整体走势", "关键走势", "趋势研判", "综合研判", "综合判定", "量价关系",
		"技术特征", "操作建议", "风险提示", "后市展望", "趋势演变",
	} {
		if strings.Contains(title, kw) {
			return true
		}
	}
	return false
}

func isIntradayRawDataLine(line string) bool {
	trim := strings.TrimSpace(strings.TrimLeft(line, "- "))
	if trim == "" {
		return false
	}
	if intradayRawOHLCHeaderRE.MatchString(trim) {
		return true
	}
	if intradayRawStatsHeaderRE.MatchString(trim) {
		return true
	}
	if intradayRawOHLCRowRE.MatchString(trim) {
		return true
	}
	if intradayKlineDetailRowRE.MatchString(trim) {
		return true
	}
	colons := strings.Count(trim, "：") + strings.Count(trim, ":")
	if colons >= 5 && regexp.MustCompile(`\d`).MatchString(trim) {
		return true
	}
	if strings.Contains(trim, "特征：表现：含义") {
		return true
	}
	if strings.Contains(trim, "K线形态") && strings.Contains(trim, "信号") {
		return true
	}
	return false
}

func polishIntradayVerdictLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	if isIntradayRawDataLine(line) {
		return ""
	}
	chunks := regexp.MustCompile(`\s*>\s*`).Split(strings.TrimPrefix(line, ">"), -1)
	parts := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		chunk = strings.TrimSpace(chunk)
		chunk = regexp.MustCompile(`^#{1,6}\s*`).ReplaceAllString(chunk, "")
		chunk = regexp.MustCompile(`^[⚠️💡📊✅🔍⬆️⬇️🔴🟢🟡]+\s*`).ReplaceAllString(chunk, "")
		chunk = strings.Trim(chunk, "* ")
		if chunk == "" {
			continue
		}
		if idx := strings.Index(chunk, "："); idx > 0 && idx < 24 {
			label := strings.TrimSpace(chunk[:idx])
			text := strings.TrimSpace(chunk[idx+len("："):])
			if label != "" && text != "" {
				parts = append(parts, "**"+label+"**："+text)
				continue
			}
		}
		parts = append(parts, chunk)
	}
	return strings.Join(parts, "\n\n")
}

func formatIntradaySection(raw, fallback string, signalOnly bool) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	if c := ExtractTaggedConclusion(raw); c != "" && signalOnly {
		return PolishStockPremarketMarkdown("**综合判定**\n\n" + polishIntradayVerdictLine(c))
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
		p = strings.Trim(RepairMCPBoldArtifacts(p), "# ")
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
				buf = append(buf, "**"+strings.TrimSuffix(title, "：")+"**")
			}
			continue
		}
		if isIntradayNoiseLine(trim) || isIntradayRawDataLine(trim) {
			continue
		}
		if signalOnly && intradayMetricRowRE.MatchString(trim) && !intradaySectionHeadRE.MatchString(trim) {
			continue
		}
		trim = strings.TrimPrefix(strings.TrimPrefix(trim, "- "), "• ")
		trim = RepairMCPBoldArtifacts(strings.TrimSpace(trim))
		if trim == "" {
			continue
		}
		buf = append(buf, trim)
	}
	flush()
	maxBlocks := 3
	if !signalOnly {
		maxBlocks = 8
	}
	if len(blocks) > maxBlocks {
		blocks = blocks[:maxBlocks]
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
	if isIntradayRawDataLine(trim) {
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
