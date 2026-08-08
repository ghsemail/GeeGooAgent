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
)

// FormatStockNews turns raw news text into a digest + headline bullets (no code, no links).
func FormatStockNews(raw, code string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.Contains(raw, "暂无") {
		return "暂无个股新闻。"
	}
	lines := strings.Split(raw, "\n")
	titles := make([]string, 0, 8)
	snippets := make([]string, 0, 8)
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
		if strings.HasPrefix(line, "🕐") {
			continue
		}
		if strings.Contains(line, code) && strings.HasPrefix(line, "##") {
			continue
		}
		title := strings.Trim(line, "*# ")
		if strings.HasPrefix(line, "**") && strings.HasSuffix(line, "**") {
			title = strings.Trim(line, "*")
		}
		if title == "" {
			continue
		}
		if strings.HasPrefix(line, "**") {
			titles = append(titles, title)
			continue
		}
		if len(title) > 12 && !strings.Contains(title, "=") {
			snippets = append(snippets, title)
		}
	}
	if len(titles) == 0 {
		clean := PolishStockPremarketMarkdown(raw)
		if clean == "" {
			return "暂无个股新闻。"
		}
		return clean
	}
	digest := buildNewsDigest(titles, snippets)
	var b strings.Builder
	b.WriteString("**新闻综述**：")
	b.WriteString(digest)
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

func buildNewsDigest(titles, snippets []string) string {
	parts := make([]string, 0, 3)
	for i := 0; i < len(titles) && len(parts) < 3; i++ {
		t := oneLine(titles[i], 48)
		if t != "" {
			parts = append(parts, t)
		}
	}
	if len(parts) == 0 && len(snippets) > 0 {
		return oneLine(snippets[0], 120)
	}
	if len(parts) == 0 {
		return "近期相关资讯较少，建议结合盘面与公告进一步确认。"
	}
	return strings.Join(parts, "；") + "。"
}

// FormatCapitalFlowSummary renders main capital flow for reports.
func FormatCapitalFlowSummary(latest map[string]any, period string) string {
	if latest == nil {
		return "暂无主力资金数据。"
	}
	main := floatFromAny(latest["main_in_flow"])
	total := floatFromAny(latest["in_flow"])
	if main == 0 && total == 0 {
		return "暂无主力资金数据。"
	}
	periodLabel := capitalPeriodLabel(period)
	mainText := flowDirectionText(main)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s主力资金**%s**", periodLabel, mainText))
	if total != 0 && (main == 0 || math.Abs(total-main) > 1) {
		b.WriteString(fmt.Sprintf("，总流入**%s**", flowDirectionText(total)))
	}
	b.WriteString("。")
	return b.String()
}

// FormatCapitalDistributionSummary renders order-size distribution for reports.
func FormatCapitalDistributionSummary(
	inSuper, inBig, inMid, inSmall,
	outSuper, outBig, outMid, outSmall float64,
	updateTime string,
) string {
	if inSuper == 0 && inBig == 0 && inMid == 0 && inSmall == 0 &&
		outSuper == 0 && outBig == 0 && outMid == 0 && outSmall == 0 {
		return "暂无资金分布数据。"
	}
	inTotal := inSuper + inBig + inMid + inSmall
	outTotal := outSuper + outBig + outMid + outSmall
	var b strings.Builder
	b.WriteString("**流入结构**：")
	b.WriteString(distributionLine(inSuper, inBig, inMid, inSmall))
	b.WriteString("\n\n**流出结构**：")
	b.WriteString(distributionLine(outSuper, outBig, outMid, outSmall))
	if inTotal > 0 || outTotal > 0 {
		net := inTotal - outTotal
		b.WriteString("\n\n**净倾向**：")
		b.WriteString(flowDirectionText(net))
		b.WriteString("（超大单")
		b.WriteString(signedMoneyCN(inSuper - outSuper))
		b.WriteString("，大单")
		b.WriteString(signedMoneyCN(inBig - outBig))
		b.WriteString("）")
	}
	if t := strings.TrimSpace(updateTime); t != "" {
		b.WriteString("\n\n*数据截至 ")
		b.WriteString(t)
		b.WriteString("*")
	}
	return b.String()
}

func distributionLine(super, big, mid, small float64) string {
	return fmt.Sprintf(
		"超大单 %s · 大单 %s · 中单 %s · 小单 %s",
		signedMoneyCN(super), signedMoneyCN(big), signedMoneyCN(mid), signedMoneyCN(small),
	)
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

// PolishStockPremarketMarkdown cleans numbers and stray metadata in report bodies.
func PolishStockPremarketMarkdown(md string) string {
	s := strings.TrimSpace(md)
	if s == "" {
		return s
	}
	s = linkLineRE.ReplaceAllString(s, "")
	s = urlLineRE.ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?m)^\s*🕐\s*.+$`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?m)^\s*\*\*?\s*(main_in_flow|in_flow|super_in|big_in|mid_in|small_in|update_time)\s*=\s*[^*\n]+\*?\*?\s*$`).ReplaceAllString(s, "")
	s = HumanizeScientificNumbers(s)
	s = regexp.MustCompile(`\n{3,}`).ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

// HumanizeScientificNumbers replaces 1.23e8 style literals with readable amounts.
func HumanizeScientificNumbers(text string) string {
	return scientificNumRE.ReplaceAllStringFunc(text, func(match string) string {
		v, err := strconv.ParseFloat(match, 64)
		if err != nil {
			return match
		}
		if math.Abs(v) >= 1e4 || (math.Abs(v) > 0 && math.Abs(v) < 0.01) {
			return FormatMoneyCN(v)
		}
		return trimFloat(v)
	})
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

func signedMoneyCN(v float64) string {
	if v == 0 {
		return "持平"
	}
	return FormatMoneyCN(v)
}

func flowDirectionText(v float64) string {
	if v > 0 {
		return "净流入 " + FormatMoneyCN(v)
	}
	if v < 0 {
		return "净流出 " + FormatMoneyCN(-v)
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

func floatFromAny(v any) float64 {
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

func oneLine(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if max > 0 && len([]rune(s)) > max {
		rs := []rune(s)
		return string(rs[:max]) + "…"
	}
	return s
}
