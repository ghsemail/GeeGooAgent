package stockfmt

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var (
	stockNewsCodeHeaderRE = regexp.MustCompile(`(?m)^##\s*【个股新闻[^\n]*】\s*$`)
	linkLineRE            = regexp.MustCompile(`(?m)^\s*🔗\s*.+$`)
	urlLineRE             = regexp.MustCompile(`(?m)^\s*https?://\S+\s*$`)
	scientificNumRE       = regexp.MustCompile(`(-?\d+(?:\.\d+)?)[eE]([+-]?\d+)`)
	mdTableSepRE          = regexp.MustCompile(`(?m)^\|?[\s:-]+\|[\s|:-]+$`)
	newsNumberPrefixRE    = regexp.MustCompile(`^\d+[\.\)、]?\s*`)
	evidenceRefRE         = regexp.MustCompile(`\[ev_[^\]]*\]`)
	bareEvidenceRE        = regexp.MustCompile(`\bev_[0-9a-zA-Z]{4,}\b`)
	corruptEvidenceRE     = regexp.MustCompile(`ev_[0-9a-zA-Z.+亿\-]{6,}`)
	sectionHeadingRE      = regexp.MustCompile(`(?m)^##\s+`)
)

// FormatStockNews turns raw news into a digest paragraph plus clean headline bullets.
func FormatStockNews(raw, code, stockName string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.Contains(raw, "暂无") {
		return "暂无个股新闻。"
	}
	lines := strings.Split(raw, "\n")
	titles := make([]string, 0, 8)
	seen := map[string]bool{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if stockNewsCodeHeaderRE.MatchString(line) {
			continue
		}
		if linkLineRE.MatchString(line) || urlLineRE.MatchString(line) {
			continue
		}
		if strings.HasPrefix(line, "🕐") || strings.HasPrefix(line, "**新闻综述**") {
			continue
		}
		if strings.Contains(line, code) && strings.HasPrefix(line, "##") {
			continue
		}
		title := normalizeNewsTitle(line, stockName)
		if title == "" || strings.Contains(title, "=") {
			continue
		}
		if seen[title] {
			continue
		}
		seen[title] = true
		titles = append(titles, title)
	}
	if len(titles) == 0 {
		clean := PolishStockNewsSection(PolishStockPremarketMarkdown(raw), stockName)
		if clean == "" {
			return "暂无个股新闻。"
		}
		return clean
	}
	var b strings.Builder
	b.WriteString("**新闻综述**：")
	b.WriteString(buildNewsDigest(titles))
	b.WriteString("\n\n")
	limit := len(titles)
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		b.WriteString("- ")
		b.WriteString(titles[i])
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func normalizeNewsTitle(line, stockName string) string {
	title := strings.TrimSpace(line)
	title = strings.TrimLeft(title, "-*# ")
	if strings.HasPrefix(line, "**") && strings.HasSuffix(line, "**") {
		title = strings.Trim(title, "*")
	}
	title = newsNumberPrefixRE.ReplaceAllString(title, "")
	title = stripCompanyPrefix(title, stockName)
	if idx := strings.Index(title, ":"); idx > 0 && idx < 16 {
		title = strings.TrimSpace(title[idx+1:])
	}
	if idx := strings.Index(title, "："); idx > 0 && idx < 16 {
		title = strings.TrimSpace(title[idx+len("："):])
	}
	title = newsNumberPrefixRE.ReplaceAllString(title, "")
	return strings.TrimSpace(title)
}

func stripCompanyPrefix(title, company string) string {
	company = strings.TrimSpace(company)
	if company == "" {
		return title
	}
	for _, sep := range []string{":", "："} {
		prefix := company + sep
		if strings.HasPrefix(title, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(title, prefix))
		}
	}
	return title
}

func buildNewsDigest(titles []string) string {
	if len(titles) == 0 {
		return "近期相关资讯较少，建议结合盘面与公告进一步确认。"
	}
	themes := extractNewsThemes(titles)
	if len(themes) > 0 {
		return fmt.Sprintf("共 %d 条相关资讯，主要涉及%s。", len(titles), strings.Join(themes, "、"))
	}
	return fmt.Sprintf("共 %d 条相关资讯，建议结合公告与盘面进一步确认。", len(titles))
}

func extractNewsThemes(titles []string) []string {
	keywords := []struct {
		key string
		tag string
	}{
		{"增持", "股东增持"},
		{"分派", "权益分派"},
		{"分红", "分红派息"},
		{"回购", "股份回购"},
		{"业绩", "业绩披露"},
		{"公告", "公司公告"},
		{"研报", "机构研报"},
		{"中标", "订单中标"},
		{"合作", "战略合作"},
	}
	seen := map[string]bool{}
	out := make([]string, 0, 4)
	for _, t := range titles {
		for _, kw := range keywords {
			if strings.Contains(t, kw.key) && !seen[kw.tag] {
				seen[kw.tag] = true
				out = append(out, kw.tag)
			}
		}
	}
	if len(out) > 4 {
		out = out[:4]
	}
	return out
}

// FormatCapitalFlowSummary renders main capital flow for reports.
func FormatCapitalFlowSummary(latest map[string]any, period string) string {
	if latest == nil {
		return "暂无主力资金数据。"
	}
	main := FloatFromAny(latest["main_in_flow"])
	total := FloatFromAny(latest["in_flow"])
	if main == 0 && total == 0 {
		return "暂无主力资金数据。"
	}
	periodLabel := capitalPeriodLabel(period)
	mainText := flowDirectionText(main)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s主力资金**%s**", periodLabel, mainText))
	if total != 0 && (main == 0 || math.Abs(total-main) > 1) {
		b.WriteString(fmt.Sprintf("，整体**%s**", flowDirectionText(total)))
	}
	b.WriteString("。")
	return b.String()
}

// FormatCapitalSection renders capital flow + distribution with a brief interpretation.
func FormatCapitalSection(
	mainIn, totalIn,
	inSuper, inBig, inMid, inSmall,
	outSuper, outBig, outMid, outSmall float64,
	updateTime string,
) string {
	if mainIn == 0 && totalIn == 0 && inSuper == 0 && inBig == 0 && inMid == 0 && inSmall == 0 &&
		outSuper == 0 && outBig == 0 && outMid == 0 && outSmall == 0 {
		return "暂无资金数据。"
	}
	superNet := inSuper - outSuper
	bigNet := inBig - outBig
	midNet := inMid - outMid
	smallNet := inSmall - outSmall

	var b strings.Builder
	if mainIn != 0 || totalIn != 0 {
		b.WriteString("**资金概况**：")
		if mainIn != 0 {
			b.WriteString("主力")
			b.WriteString(flowDirectionText(mainIn))
		}
		if totalIn != 0 && (mainIn == 0 || math.Abs(totalIn-mainIn) > 1) {
			if mainIn != 0 {
				b.WriteString("，整体")
			} else {
				b.WriteString("整体")
			}
			b.WriteString(flowDirectionText(totalIn))
		}
		b.WriteString("。\n\n")
	}
	b.WriteString("**简要解读**：")
	b.WriteString(capitalAnalysisText(mainIn, superNet, bigNet, midNet, smallNet))
	b.WriteString("\n\n")
	b.WriteString("**分单结构**：超大单净")
	b.WriteString(signedMoneyCN(superNet))
	b.WriteString(" · 大单净")
	b.WriteString(signedMoneyCN(bigNet))
	b.WriteString(" · 中单净")
	b.WriteString(signedMoneyCN(midNet))
	b.WriteString(" · 小单净")
	b.WriteString(signedMoneyCN(smallNet))
	if t := strings.TrimSpace(updateTime); t != "" {
		b.WriteString("\n\n*数据截至 ")
		b.WriteString(t)
		b.WriteString("*")
	}
	return b.String()
}

func capitalAnalysisText(mainIn, superNet, bigNet, midNet, smallNet float64) string {
	parts := make([]string, 0, 3)
	switch {
	case mainIn > 0:
		parts = append(parts, "主力资金呈净流入，短线资金面偏多")
	case mainIn < 0:
		parts = append(parts, "主力资金呈净流出，短线资金面偏空")
	default:
		parts = append(parts, "主力资金变动不大，需结合量价判断")
	}
	if superNet > 0 && bigNet > 0 {
		parts = append(parts, "超大单与大单同步净流入，机构参与度提升")
	} else if superNet < 0 && bigNet < 0 {
		parts = append(parts, "超大单与大单同步净流出，主力减仓迹象明显")
	} else if superNet < 0 && bigNet > 0 {
		parts = append(parts, "超大单净流出、大单净流入，分歧加大")
	} else if superNet > 0 && bigNet < 0 {
		parts = append(parts, "超大单净流入、大单净流出，筹码结构分化")
	}
	if smallNet > 0 && mainIn <= 0 {
		parts = append(parts, "小单净流入而主力偏弱，散户接盘特征需警惕")
	}
	if len(parts) == 0 {
		return "资金结构整体平稳，建议结合盘中量价验证。"
	}
	return strings.Join(parts, "；") + "。"
}

// FormatCapitalDistributionSummary renders order-size distribution for reports.
func FormatCapitalDistributionSummary(
	inSuper, inBig, inMid, inSmall,
	outSuper, outBig, outMid, outSmall float64,
	updateTime string,
) string {
	return FormatCapitalSection(0, 0, inSuper, inBig, inMid, inSmall, outSuper, outBig, outMid, outSmall, updateTime)
}

// FormatWeeklyAnalysis normalizes MCP weekly output for stock reports.
func FormatWeeklyAnalysis(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "暂无周线技术分析。"
	}
	s := raw
	s = regexp.MustCompile(`(?m)^#\s+.+$`).ReplaceAllString(s, "")
	s = mdTableSepRE.ReplaceAllString(s, "")
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "|") && strings.Contains(trim, "|") {
			cells := splitTableRow(trim)
			if len(cells) >= 2 && !isTableHeaderRow(cells) {
				out = append(out, "- "+strings.Join(cells, "："))
			}
			continue
		}
		if strings.HasPrefix(trim, "## ") {
			trim = "### " + strings.TrimPrefix(trim, "## ")
		}
		out = append(out, trim)
	}
	s = strings.Join(out, "\n")
	s = PolishStockPremarketMarkdown(s)
	if strings.TrimSpace(s) == "" {
		return "暂无周线技术分析。"
	}
	return s
}

// FormatEmbeddedHourlyAnalysis normalizes MCP hourly blocks for embedding under 今日行情.
func FormatEmbeddedHourlyAnalysis(raw string) string {
	s := FormatWeeklyAnalysis(raw)
	if strings.TrimSpace(s) == "" || s == "暂无周线技术分析。" {
		return s
	}
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			out = append(out, "")
			continue
		}
		if trim == "---" {
			continue
		}
		if isGarbageTableLine(trim) {
			continue
		}
		if isIntradayRawDataLine(trim) {
			continue
		}
		if strings.HasPrefix(trim, "```") {
			continue
		}
		if strings.HasPrefix(trim, ">") {
			inner := strings.TrimSpace(strings.TrimPrefix(trim, ">"))
			inner = strings.Trim(inner, "* ")
			if looksTruncatedFragment(inner) || looksIncompleteBlockquote(inner) {
				continue
			}
		}
		if looksTruncatedFragment(strings.Trim(trim, "*# ")) {
			continue
		}
		if strings.HasPrefix(trim, "### ") {
			title := strings.TrimSpace(strings.TrimPrefix(trim, "### "))
			if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
				out = append(out, "")
			}
			out = append(out, "**"+title+"**")
			continue
		}
		if title := strings.TrimLeft(trim, "# "); cnSectionRE.MatchString(title) {
			if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
				out = append(out, "")
			}
			out = append(out, "**"+title+"**")
			continue
		}
		if strings.HasPrefix(trim, "- ") || strings.HasPrefix(trim, "• ") {
			trim = strings.TrimLeft(strings.TrimPrefix(strings.TrimPrefix(trim, "- "), "• "), " ")
		}
		out = append(out, trim)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func isGarbageTableLine(line string) bool {
	switch strings.TrimSpace(line) {
	case "- 方向：数量：占比",
		"- 日期：收盘价：较前一日涨跌幅：备注",
		"- 日期：成交量（手）：备注",
		"- 阶段：时间区间：趋势特征",
		"- 日期：形态名称：中文名称：信号":
		return true
	}
	return false
}

// FormatHourlySignalAnalysis normalizes hourly signal MCP output; strips raw 0/1/-1 codes.
func FormatHourlySignalAnalysis(raw string) string {
	s := FormatEmbeddedHourlyAnalysis(raw)
	if strings.TrimSpace(s) == "" || s == "暂无周线技术分析。" {
		return "暂无小时级信号分析。"
	}
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			out = append(out, "")
			continue
		}
		if polished := polishSignalMetricLine(trim); polished != "" {
			out = append(out, polished)
		}
	}
	return strings.Join(out, "\n")
}

func polishSignalMetricLine(line string) string {
	trim := strings.TrimSpace(line)
	if trim == "" {
		return ""
	}
	if m := signalMetricLineRE.FindStringSubmatch(trim); len(m) == 4 {
		prefix := m[1]
		label := strings.TrimSpace(m[2])
		verdict := strings.TrimSpace(strings.Trim(m[3], "*"))
		if label != "" && verdict != "" {
			return prefix + label + "：" + verdict
		}
	}
	return trim
}

func looksTruncatedFragment(s string) bool {
	s = strings.TrimSpace(strings.Trim(s, "*"))
	if s == "" {
		return false
	}
	if matched, _ := regexp.MatchString(`^\d+\.\s*\*\*[^*]+$`, s); matched {
		return true
	}
	runes := []rune(s)
	if len(runes) < 48 && !strings.ContainsAny(s, "。！？.") {
		if matched, _ := regexp.MatchString(`^\d+\.\s+`, s); matched {
			return true
		}
	}
	return false
}

func looksIncompleteBlockquote(s string) bool {
	s = strings.TrimSpace(strings.Trim(s, "*"))
	if s == "" {
		return false
	}
	if strings.Contains(s, "（") && !strings.Contains(s, "）") {
		return true
	}
	if strings.HasSuffix(s, ".") && !strings.ContainsAny(s, "。！？") {
		return true
	}
	return false
}

var (
	signalBinaryRE     = regexp.MustCompile(`[：:]\s*0\s*$`)
	signalMetricCodeRE = regexp.MustCompile(`[：:]\s*[-+]?\d+\s*：\s*\*\*([^*]+)\*\*`)
	signalMetricLineRE = regexp.MustCompile(`^(-\s*)?(.+?)[：:]\s*[-+]?\d+\s*[：:]\s*(?:\*\*)?([^*\n]+?)(?:\*\*)?\s*$`)
	cnSectionRE        = regexp.MustCompile(`^[一二三四五六七八九十]+、`)
)

// PostmarketSummaryExcerpt returns the first readable paragraph from hourly MCP markdown.
func PostmarketSummaryExcerpt(raw string, maxRunes int) string {
	s := FormatWeeklyAnalysis(raw)
	if strings.TrimSpace(s) == "" || s == "暂无周线技术分析。" {
		return ""
	}
	var para []string
	for _, line := range strings.Split(s, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || trim == "---" {
			if len(para) > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(trim, "#") {
			if len(para) > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(trim, "- ") && signalMetricCodeRE.MatchString(trim) {
			continue
		}
		para = append(para, trim)
		if len([]rune(strings.Join(para, ""))) >= maxRunes {
			break
		}
	}
	text := strings.TrimSpace(strings.Join(para, " "))
	if text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) > maxRunes {
		text = string(runes[:maxRunes]) + "…"
	}
	return text
}

// ExtractTaggedConclusion pulls a one-line verdict from MCP hourly analysis.
func ExtractTaggedConclusion(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.Contains(line, "整体结论") && !strings.Contains(line, "整体判断") {
			continue
		}
		line = strings.TrimPrefix(line, "> ")
		line = strings.Trim(line, "*# ")
		for _, sep := range []string{"：", ":"} {
			if i := strings.Index(line, sep); i >= 0 && i < len(line)-len(sep) {
				out := strings.TrimSpace(line[i+len(sep):])
				out = strings.Trim(out, "* ")
				if out != "" && !looksTruncatedFragment(out) {
					return out
				}
			}
		}
	}
	return ""
}

// FirstSentence returns the first complete sentence from plain text.
func FirstSentence(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	runes := []rune(text)
	for i, r := range runes {
		if r == '。' || r == '！' || r == '？' {
			return strings.TrimSpace(string(runes[:i+1]))
		}
	}
	if len(runes) > 120 {
		return strings.TrimSpace(string(runes[:120])) + "…"
	}
	return text
}

func ReplaceMarkdownSection(md, heading, body string) string {
	heading = strings.TrimSpace(heading)
	body = strings.TrimSpace(body)
	if heading == "" || body == "" {
		return md
	}
	re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(heading) + `\s*$`)
	loc := re.FindStringIndex(md)
	if loc == nil {
		return strings.TrimSpace(md + "\n\n" + heading + "\n\n" + body)
	}
	start := loc[1]
	rest := md[start:]
	next := sectionHeadingRE.FindStringIndex(rest)
	end := len(md)
	if next != nil && next[0] > 0 {
		end = start + next[0]
	}
	tail := strings.TrimSpace(md[end:])
	if tail == "" {
		return strings.TrimSpace(md[:loc[0]] + heading + "\n\n" + body)
	}
	return strings.TrimSpace(md[:loc[0]] + heading + "\n\n" + body + "\n\n" + tail)
}

// PolishStockNewsSection rebuilds digest and strips numbered prefixes from headlines.
func PolishStockNewsSection(section, stockName string) string {
	section = strings.TrimSpace(section)
	if section == "" {
		return section
	}
	titles := make([]string, 0, 8)
	seen := map[string]bool{}
	for _, line := range strings.Split(section, "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "**新闻综述**") {
			continue
		}
		if strings.HasPrefix(t, "-") {
			t = strings.TrimSpace(strings.TrimPrefix(t, "-"))
		}
		title := normalizeNewsTitle(t, stockName)
		if title == "" {
			continue
		}
		if seen[title] {
			continue
		}
		seen[title] = true
		titles = append(titles, title)
	}
	if len(titles) == 0 {
		return section
	}
	var b strings.Builder
	b.WriteString("**新闻综述**：")
	b.WriteString(buildNewsDigest(titles))
	b.WriteString("\n\n")
	limit := len(titles)
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		b.WriteString("- ")
		b.WriteString(titles[i])
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

// PolishStockNewsInReport fixes the ## 个股新闻 section inside a full report.
func PolishStockNewsInReport(md, stockName string) string {
	re := regexp.MustCompile(`(?m)^##\s+个股新闻\s*$`)
	loc := re.FindStringIndex(md)
	if loc == nil {
		return md
	}
	start := loc[1]
	rest := md[start:]
	next := sectionHeadingRE.FindStringIndex(rest)
	end := len(md)
	if next != nil && next[0] > 0 {
		end = start + next[0]
	}
	body := PolishStockNewsSection(strings.TrimSpace(md[start:end]), stockName)
	return ReplaceMarkdownSection(md, "## 个股新闻", body)
}

// PolishStockPremarketMarkdown cleans numbers and stray metadata in report bodies.
func PolishStockPremarketMarkdown(md string) string {
	s := strings.TrimSpace(md)
	if s == "" {
		return s
	}
	s = StripEvidenceRefs(s)
	s = linkLineRE.ReplaceAllString(s, "")
	s = urlLineRE.ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?m)^\s*🕐\s*.+$`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?m)^\s*\*\*?\s*(main_in_flow|in_flow|super_in|big_in|mid_in|small_in|update_time)\s*=\s*[^*\n]+\*?\*?\s*$`).ReplaceAllString(s, "")
	s = HumanizeScientificNumbers(s)
	s = localizeAttitudeInText(s)
	s = RepairMCPBoldArtifacts(s)
	s = regexp.MustCompile(`\n{3,}`).ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

// RepairMCPBoldArtifacts fixes broken MCP bold markers while keeping intentional **titles**.
func RepairMCPBoldArtifacts(s string) string {
	s = regexp.MustCompile(`\*\*([^*\n]*?[+-]?\d+(?:\.\d+)?%[^*\n]*?)\*\*`).ReplaceAllString(s, "$1")
	s = regexp.MustCompile(`\*\*([^*\n]{1,48}：[+-]?\d+(?:\.\d+)?%[^\n]*)`).ReplaceAllString(s, "$1")
	s = regexp.MustCompile(`([+-]?\d+(?:\.\d+)?%)\*\*`).ReplaceAllString(s, "$1")
	s = regexp.MustCompile(`\*\*([+-]?\d+(?:\.\d+)?%)`).ReplaceAllString(s, "$1")
	s = strings.ReplaceAll(s, "****", "**")
	return s
}

// RepairIntradayLineBreaks fixes glued list items and section headers in hourly prose.
func RepairIntradayLineBreaks(md string) string {
	s := strings.TrimSpace(md)
	if s == "" {
		return s
	}
	repls := []struct{ re, neu string }{
		{`([。！？])\s*(\d+\.\s+\*\*)`, "$1\n\n$2"},
		{`([。！？])\s*(\*\*[一二三四五六七八九十]+、)`, "$1\n\n$2"},
		{`([。！？])\s*(\*\*[^*\n]{2,24}\*\*)`, "$1\n\n$2"},
		{`([）)])\s*(\d{1,2}/\d{1,2}[：:])`, "$1\n$2"},
		{`([）)])\s*(\d{1,2}月\d{1,2}日)`, "$1\n$2"},
		{`(\S)\s{0,2}(8月[5-9]日)`, "$1\n\n$2"},
	}
	for _, r := range repls {
		s = regexp.MustCompile(r.re).ReplaceAllString(s, r.neu)
	}
	s = regexp.MustCompile(`\n{3,}`).ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

// StripBrokenBoldMarkers is an alias kept for callers that only need MCP artifact repair.
func StripBrokenBoldMarkers(s string) string {
	return RepairMCPBoldArtifacts(s)
}

// FormatPrice formats a stock quote without trailing zeros.
func FormatPrice(v float64) string {
	s := fmt.Sprintf("%.4f", v)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	return s
}

// FormatQty formats share counts without unnecessary decimals.
func FormatQty(v float64) string {
	if math.Mod(v, 1) == 0 {
		return fmt.Sprintf("%.0f", v)
	}
	return trimFloat(v)
}

// FormatPercent formats a percentage value for report prose.
func FormatPercent(v float64) string {
	sign := ""
	if v > 0 {
		sign = "+"
	}
	return sign + trimFloat(v) + "%"
}

// FormatSignedMoneyCN formats signed CNY amounts for position P/L prose.
func FormatSignedMoneyCN(v float64) string {
	if v == 0 {
		return "0"
	}
	sign := ""
	if v < 0 {
		sign = "-"
	} else {
		sign = "+"
	}
	return sign + formatMoneyAbs(v)
}

// StripEvidenceRefs removes internal evidence citation markers from user-facing text.
func StripEvidenceRefs(text string) string {
	s := evidenceRefRE.ReplaceAllString(text, "")
	s = bareEvidenceRE.ReplaceAllString(s, "")
	s = corruptEvidenceRE.ReplaceAllString(s, "")
	s = regexp.MustCompile(`\(\s*\)`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`[ \t]{2,}`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`[，,；;]\s*[，,；;]+`).ReplaceAllString(s, "，")
	return strings.TrimSpace(s)
}

// HumanizeScientificNumbers replaces 1.23e8 style literals with readable amounts.
func HumanizeScientificNumbers(text string) string {
	placeholders := map[string]string{}
	i := 0
	protected := corruptEvidenceRE.ReplaceAllStringFunc(text, func(m string) string {
		key := fmt.Sprintf("§EV%d§", i)
		placeholders[key] = ""
		i++
		return key
	})
	protected = evidenceRefRE.ReplaceAllStringFunc(protected, func(m string) string {
		key := fmt.Sprintf("§EV%d§", i)
		placeholders[key] = ""
		i++
		return key
	})
	protected = bareEvidenceRE.ReplaceAllStringFunc(protected, func(m string) string {
		key := fmt.Sprintf("§EV%d§", i)
		placeholders[key] = ""
		i++
		return key
	})
	out := scientificNumRE.ReplaceAllStringFunc(protected, func(match string) string {
		v, err := strconv.ParseFloat(match, 64)
		if err != nil {
			return match
		}
		if math.Abs(v) >= 1e4 || (math.Abs(v) > 0 && math.Abs(v) < 0.01) {
			return FormatMoneyCN(v)
		}
		return trimFloat(v)
	})
	for key := range placeholders {
		out = strings.ReplaceAll(out, key, "")
	}
	return out
}

// LocalizeAttitude maps bot attitude codes to Chinese labels.
func LocalizeAttitude(attitude string) string {
	switch strings.ToLower(strings.TrimSpace(attitude)) {
	case "bullish", "long":
		return "偏多"
	case "bearish", "short":
		return "偏空"
	default:
		return "中性"
	}
}

func localizeAttitudeInText(text string) string {
	repl := map[string]string{
		"**neutral**": "**中性**",
		"**bullish**": "**偏多**",
		"**bearish**": "**偏空**",
		" neutral":    " 中性",
		" bullish":    " 偏多",
		" bearish":    " 偏空",
	}
	for old, neu := range repl {
		text = strings.ReplaceAll(text, old, neu)
	}
	return text
}

// FormatMoneyCN formats CNY amounts for report prose.
func FormatMoneyCN(v float64) string {
	abs := math.Abs(v)
	sign := ""
	if v < 0 {
		sign = "-"
	} else if v > 0 {
		sign = "+"
	}
	switch {
	case abs >= 1e8:
		return sign + fmt.Sprintf("%.2f亿", abs/1e8)
	case abs >= 1e4:
		return sign + fmt.Sprintf("%.2f万", abs/1e4)
	default:
		return sign + trimFloat(abs)
	}
}

func formatMoneyAbs(v float64) string {
	abs := math.Abs(v)
	switch {
	case abs >= 1e8:
		return fmt.Sprintf("%.2f亿", abs/1e8)
	case abs >= 1e4:
		return fmt.Sprintf("%.2f万", abs/1e4)
	default:
		return trimFloat(abs)
	}
}

func signedMoneyCN(v float64) string {
	if v == 0 {
		return "持平"
	}
	if v > 0 {
		return "流入" + formatMoneyAbs(v)
	}
	return "流出" + formatMoneyAbs(v)
}

func flowDirectionText(v float64) string {
	if v > 0 {
		return "净流入 " + formatMoneyAbs(v)
	}
	if v < 0 {
		return "净流出 " + formatMoneyAbs(v)
	}
	return "持平"
}

func capitalPeriodLabel(period string) string {
	switch strings.ToUpper(strings.TrimSpace(period)) {
	case "WEEK":
		return "近一周"
	case "MONTH":
		return "近一月"
	default:
		return "当日"
	}
}

// FloatFromAny parses numeric tool payload fields.
func FloatFromAny(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f
	default:
		return 0
	}
}

func trimFloat(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	return s
}

func splitTableRow(line string) []string {
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func isTableHeaderRow(cells []string) bool {
	if len(cells) == 0 {
		return true
	}
	joined := strings.ToLower(strings.Join(cells, ""))
	return strings.Contains(joined, "指标") || strings.Contains(joined, "indicator")
}
