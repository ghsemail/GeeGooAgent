package stockdigest

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/stockfmt"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/decision"
)

var (
	markdownBoldRe   = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	markdownHeaderRe = regexp.MustCompile(`(?m)^#{1,6}\s*`)
)

func formatPremarketStock(ws memory.StockWorkspace) string {
	name := displayName(ws)
	result := localizeTerm(ws.PreMarketResult, "中性")
	conf := localizeTerm(ws.PreMarketConfidence, "—")
	suggest := localizeTerm(ws.PreMarketSuggestion, "—")
	suggestAction := premarketSuggestionPhrase(ws.PreMarketSuggestion)

	parts := []string{
		mdStockTitle(name, ws.Code),
		mdBullets(
			mdBullet("结论", result),
			mdBullet("置信度", conf),
			mdBullet("建议", suggest),
		),
	}
	if suggestAction != "" {
		parts = append(parts, mdSection("操作建议", mdQuote(suggestAction)))
	}
	if summary := premarketDigestSummary(ws); summary != "" {
		parts = append(parts, mdSection("摘要", summary))
	}
	if reason := premarketDigestReason(ws); reason != "" {
		parts = append(parts, mdSection("判定依据", formatReasonMarkdown(reason)))
	}
	return mdParagraphs(parts...)
}

func formatPostmarketStock(ws memory.StockWorkspace) string {
	name := displayName(ws)
	var metrics []string
	if ws.ChangePct != 0 {
		sign := "+"
		if ws.ChangePct < 0 {
			sign = ""
		}
		metrics = append(metrics, mdBullet("涨跌", fmt.Sprintf("%s%.2f%%", sign, ws.ChangePct)))
	}
	if bias := decision.LocalizeSessionBias(ws.SessionBias); bias != "" {
		metrics = append(metrics, mdBullet("盘面", bias))
	}
	if vs := decision.LocalizeVsPreMarket(ws.VsPreMarket); vs != "" {
		metrics = append(metrics, mdBullet("对照盘前", vs))
	}
	parts := []string{mdStockTitle(name, ws.Code)}
	if bullets := mdBullets(metrics...); bullets != "" {
		parts = append(parts, bullets)
	}
	if body := mdSectionBody(ws.ReportSummary); body != "" {
		parts = append(parts, mdSection("摘要", body))
	}
	if body := mdSectionBody(ws.ReportMarketSummary); body != "" {
		parts = append(parts, mdSection("盘面回顾", body))
	}
	if body := mdSectionBody(ws.ReportTradeSummary); body != "" {
		parts = append(parts, mdSection("交易复盘", body))
	}
	if body := mdSectionBody(ws.ReportExperienceSummary); body != "" {
		parts = append(parts, mdSection("经验总结", body))
	}
	return mdParagraphs(parts...)
}

func mdSectionBody(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	return cleanPlainText(stockfmt.LocalizeDecisionTerms(body))
}

func premarketSuggestionPhrase(suggestion string) string {
	switch strings.ToLower(strings.TrimSpace(suggestion)) {
	case "buy":
		return "可考虑买入或加仓，但需结合盘中量价与大盘联动确认。"
	case "sell":
		return "可考虑减仓或观望，避免在弱势结构中逆势加仓。"
	case "hold":
		return "建议观望，等待方向明朗或关键位确认后再行动。"
	default:
		return ""
	}
}

func premarketDigestSummary(ws memory.StockWorkspace) string {
	raw := strings.TrimSpace(ws.ReportSummary)
	if raw != "" && !isReportExcerptSummary(raw) {
		return cleanPlainText(stockfmt.LocalizeDecisionTerms(raw))
	}
	return buildPremarketOneLiner(ws)
}

func premarketDigestReason(ws memory.StockWorkspace) string {
	raw := strings.TrimSpace(ws.PreMarketReason)
	if isBoilerplateReason(raw) {
		return buildSubstantiveReasonDigest(ws)
	}
	return cleanPlainText(stockfmt.LocalizeDecisionTerms(raw))
}

func buildPremarketOneLiner(ws memory.StockWorkspace) string {
	name := displayName(ws)
	dir := localizeTerm(ws.PreMarketResult, "中性")
	action := premarketSuggestionPhrase(ws.PreMarketSuggestion)
	parts := []string{fmt.Sprintf("%s盘前研判%s", name, dir)}
	if action != "" {
		parts = append(parts, strings.TrimSuffix(action, "。"))
	}
	if cap := capitalBrief(ws); cap != "" {
		parts = append(parts, cap)
	}
	return cleanPlainText(strings.Join(parts, "，") + "。")
}

func buildSubstantiveReasonDigest(ws memory.StockWorkspace) string {
	var parts []string
	if att := strings.TrimSpace(ws.Attitude); att != "" {
		parts = append(parts, "Bot 昨日态度为 "+stockfmt.LocalizeAttitude(att))
	}
	if cap := capitalBrief(ws); cap != "" {
		parts = append(parts, cap)
	}
	if news := newsBrief(ws.StockNewsSummary); news != "" {
		parts = append(parts, news)
	}
	if len(parts) == 0 {
		return ""
	}
	return cleanPlainText(strings.Join(parts, "；") + "。")
}

func capitalBrief(ws memory.StockWorkspace) string {
	for _, raw := range []string{ws.CapitalDistributionSummary, ws.CapitalFlowSummary} {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		for _, line := range strings.Split(raw, "\n") {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "简要解读") {
				line = strings.TrimPrefix(line, "**简要解读**：")
				return cleanPlainText(line)
			}
		}
		line := strings.Split(raw, "\n")[0]
		return cleanPlainText(line)
	}
	return ""
}

func newsBrief(summary string) string {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return ""
	}
	for _, line := range strings.Split(summary, "\n") {
		line = cleanPlainText(line)
		if strings.Contains(line, "新闻") || strings.Contains(line, "条") {
			if len([]rune(line)) > 120 {
				line = string([]rune(line)[:117]) + "…"
			}
			return line
		}
	}
	return ""
}

func isBoilerplateReason(reason string) bool {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return true
	}
	return strings.Contains(reason, "证据已纳入") && strings.Contains(reason, "条证据引用")
}

func isReportExcerptSummary(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	for _, marker := range []string{"市场背景", "新闻综述", "个股新闻", "## ", "**"} {
		if strings.Contains(s, marker) {
			return true
		}
	}
	if strings.Contains(s, "…") || strings.Contains(s, "...") {
		return true
	}
	return len([]rune(s)) > 320
}

func localizeTerm(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	switch strings.ToLower(v) {
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
	case "neutral":
		return "中性"
	case "bullish":
		return "偏多"
	case "bearish":
		return "偏空"
	default:
		return v
	}
}

func displayName(ws memory.StockWorkspace) string {
	if n := strings.TrimSpace(ws.StockName); n != "" {
		return n
	}
	return ws.Code
}

// FormatPremarketStockForTest exposes premarket digest formatting for tests.
func FormatPremarketStockForTest(ws memory.StockWorkspace) string {
	return formatPremarketStock(ws)
}

// FormatReasonMarkdownForTest exposes reason markdown formatting for tests.
func FormatReasonMarkdownForTest(reason string) string {
	return formatReasonMarkdown(reason)
}
