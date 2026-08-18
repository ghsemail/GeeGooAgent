package stockdigest

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/stockfmt"
)

var (
	evidenceMarkerRe = regexp.MustCompile(`【】`)
	reasonSemicolonRe = regexp.MustCompile(`[；;]\s*`)
)

func mdPhaseTitle(skill, market, date string) string {
	label := "个股报告"
	switch skill {
	case "premarket_stock":
		label = "盘前个股"
	case "postmarket_stock":
		label = "盘后个股"
	}
	return fmt.Sprintf("## %s · %s · %s", label, market, date)
}

func mdStockTitle(name, code string) string {
	return fmt.Sprintf("### %s（%s）", name, code)
}

func mdSection(title, body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	return fmt.Sprintf("#### %s\n\n%s", title, body)
}

func mdQuote(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, "> "+line)
	}
	return strings.Join(lines, "\n")
}

func mdBullet(label, value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "—" {
		return ""
	}
	return fmt.Sprintf("- **%s**：%s", label, value)
}

func mdBullets(items ...string) string {
	var out []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return strings.Join(out, "\n")
}

func mdParagraphs(parts ...string) string {
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, "\n\n")
}

func formatReasonMarkdown(reason string) string {
	reason = strings.TrimSpace(stockfmt.LocalizeDecisionTerms(reason))
	if reason == "" {
		return ""
	}
	if evidenceMarkerRe.MatchString(reason) {
		prefix, bullets := splitEvidenceBullets(reason)
		if len(bullets) > 0 {
			var b strings.Builder
			if prefix != "" {
				b.WriteString("**")
				b.WriteString(prefix)
				b.WriteString("**\n\n")
			}
			for _, item := range bullets {
				fmt.Fprintf(&b, "- %s\n", item)
			}
			return strings.TrimSpace(b.String())
		}
	}
	if strings.Contains(reason, "；") && len([]rune(reason)) > 120 {
		parts := reasonSemicolonRe.Split(reason, -1)
		if len(parts) >= 3 {
			prefix := strings.TrimSpace(parts[0])
			prefix = strings.TrimSuffix(prefix, "：")
			var b strings.Builder
			if prefix != "" {
				b.WriteString("**")
				b.WriteString(prefix)
				b.WriteString("**\n\n")
			}
			for _, item := range parts[1:] {
				item = strings.TrimSpace(item)
				if item != "" {
					fmt.Fprintf(&b, "- %s\n", item)
				}
			}
			return strings.TrimSpace(b.String())
		}
	}
	return reason
}

func splitEvidenceBullets(reason string) (string, []string) {
	idx := strings.Index(reason, "【】")
	if idx < 0 {
		return "", nil
	}
	prefix := strings.TrimSpace(reason[:idx])
	prefix = strings.TrimSuffix(prefix, "：")
	rest := reason[idx:]
	chunks := evidenceMarkerRe.Split(rest, -1)
	var bullets []string
	for _, chunk := range chunks {
		chunk = cleanPlainText(chunk)
		if chunk != "" {
			bullets = append(bullets, chunk)
		}
	}
	return prefix, bullets
}

func cleanPlainText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = markdownHeaderRe.ReplaceAllString(s, "")
	s = markdownBoldRe.ReplaceAllString(s, "$1")
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")
	s = strings.ReplaceAll(s, "`", "")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
