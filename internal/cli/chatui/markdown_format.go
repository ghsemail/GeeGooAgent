package chatui

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reGlueHeading   = regexp.MustCompile(`([^\n])(#{2,6}\s)`)
	reGlueCard      = regexp.MustCompile(`([^\n])(\*\*[0-9]+\.)`)
	reGlueHR        = regexp.MustCompile(`([^\n-])\s*---\s*`)
	reGlueSummary   = regexp.MustCompile(`([^\n])(小结[:：])`)
)

// NormalizeAssistantLayout inserts line breaks when the model glues markdown blocks
// onto one line (common with streaming / Chinese punctuation).
func NormalizeAssistantLayout(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	var out []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, " ")
		trim := strings.TrimSpace(line)
		if trim == "" {
			if len(out) > 0 && out[len(out)-1] != "" {
				out = append(out, "")
			}
			continue
		}
		if strings.HasPrefix(trim, "|") || isTableSeparator(trim) {
			out = append(out, line)
			continue
		}
		line = reGlueHeading.ReplaceAllString(line, "$1\n$2")
		line = reGlueCard.ReplaceAllString(line, "$1\n$2")
		line = reGlueHR.ReplaceAllString(line, "$1\n---\n")
		line = reGlueSummary.ReplaceAllString(line, "$1\n$2")
		for _, sub := range strings.Split(line, "\n") {
			sub = strings.TrimRight(sub, " ")
			if strings.TrimSpace(sub) == "" {
				continue
			}
			out = append(out, sub)
		}
	}
	text = strings.Join(out, "\n")
	return breakInlinePipeFields(text)
}

func breakInlinePipeFields(text string) string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		if strings.Count(trim, " | ") >= 2 && !strings.HasPrefix(trim, "|") {
			parts := strings.Split(trim, " | ")
			out = append(out, strings.TrimSpace(parts[0]))
			for _, p := range parts[1:] {
				p = strings.TrimSpace(p)
				if p != "" {
					out = append(out, "- "+p)
				}
			}
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// PreprocessTerminalMarkdown adapts assistant markdown for narrow terminals:
// wide pipe tables become card-style bullet blocks that glamour can wrap cleanly.
func PreprocessTerminalMarkdown(text string) string {
	text = NormalizeAssistantLayout(text)
	if !strings.Contains(text, "|") {
		return text
	}
	lines := strings.Split(text, "\n")
	var out []string
	for i := 0; i < len(lines); {
		if cells := parseTableRow(lines[i]); cells != nil {
			tableLines := []string{lines[i]}
			i++
			for i < len(lines) {
				if isTableSeparator(lines[i]) {
					tableLines = append(tableLines, lines[i])
					i++
					continue
				}
				if cells := parseTableRow(lines[i]); cells != nil {
					tableLines = append(tableLines, lines[i])
					i++
					continue
				}
				break
			}
			if formatted := formatTableBlock(tableLines); formatted != "" {
				if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
					out = append(out, "")
				}
				out = append(out, formatted)
				out = append(out, "")
			} else {
				out = append(out, tableLines...)
			}
			continue
		}
		out = append(out, lines[i])
		i++
	}
	return strings.Join(out, "\n")
}

func parseTableRow(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") {
		return nil
	}
	raw := strings.Split(line, "|")
	var cells []string
	for _, part := range raw {
		part = strings.TrimSpace(part)
		if part != "" {
			cells = append(cells, part)
		}
	}
	if len(cells) < 2 {
		return nil
	}
	return cells
}

func isTableSeparator(line string) bool {
	line = strings.TrimSpace(line)
	if !strings.Contains(line, "|") || !strings.Contains(line, "-") {
		return false
	}
	for _, r := range line {
		switch r {
		case '|', '-', ':', ' ', '\t':
		default:
			return false
		}
	}
	return true
}

func formatTableBlock(tableLines []string) string {
	if len(tableLines) < 2 {
		return ""
	}
	headers := parseTableRow(tableLines[0])
	if headers == nil {
		return ""
	}
	start := 1
	if start < len(tableLines) && isTableSeparator(tableLines[start]) {
		start++
	}
	var rows [][]string
	for _, line := range tableLines[start:] {
		if cells := parseTableRow(line); cells != nil {
			rows = append(rows, cells)
		}
	}
	if len(rows) == 0 {
		return ""
	}
	var b strings.Builder
	for i, row := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(formatTableRowCard(i+1, headers, row))
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatTableRowCard(fallbackNum int, headers, cells []string) string {
	title := ""
	code := ""
	num := fallbackNum
	var pairs []string

	for i, h := range headers {
		if i >= len(cells) {
			break
		}
		h = strings.TrimSpace(h)
		v := strings.TrimSpace(cells[i])
		if v == "" || v == "-" || v == "—" {
			continue
		}
		hl := strings.ToLower(h)
		switch {
		case hl == "#" || hl == "序号" || hl == "no" || hl == "no.":
			if n, err := fmt.Sscanf(v, "%d", &num); n == 1 && err == nil {
				continue
			}
		case strings.Contains(h, "名称") || hl == "name" || hl == "botname" || h == "名称":
			title = v
			continue
		case strings.Contains(h, "代码") || hl == "code" || h == "代码":
			code = v
			continue
		}
		pairs = append(pairs, h+"："+v)
	}
	if title == "" {
		for _, c := range cells {
			c = strings.TrimSpace(c)
			if c != "" && c != "-" && !isNumericIndex(c) {
				title = c
				break
			}
		}
	}
	if title == "" {
		title = fmt.Sprintf("条目 %d", num)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("**%d. %s**", num, title))
	if code != "" {
		b.WriteString(" · `")
		b.WriteString(code)
		b.WriteString("`")
	}
	if len(pairs) == 0 {
		return b.String()
	}
	b.WriteByte('\n')
	mid := (len(pairs) + 1) / 2
	b.WriteString("- ")
	b.WriteString(strings.Join(pairs[:mid], "  "))
	if mid < len(pairs) {
		b.WriteByte('\n')
		b.WriteString("- ")
		b.WriteString(strings.Join(pairs[mid:], "  "))
	}
	return b.String()
}

func isNumericIndex(s string) bool {
	s = strings.TrimSpace(strings.TrimSuffix(s, "."))
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// RenderPlainAssistantBody renders assistant text line-by-line when glamour is unavailable
// or during live preview (preserves newlines; avoids lipgloss width reflow on whole blocks).
func RenderPlainAssistantBody(text string) string {
	text = PreprocessTerminalMarkdown(text)
	if strings.TrimSpace(text) == "" {
		return styleDim.Render("⋯ 正在生成回复…")
	}
	var b strings.Builder
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, " ")
		if line == "" {
			b.WriteByte('\n')
			continue
		}
		trim := strings.TrimLeft(line, " ")
		indent := line[:len(line)-len(trim)]
		switch {
		case strings.HasPrefix(trim, "## "):
			b.WriteString(indent + styleGold.Render(trim) + "\n")
		case strings.HasPrefix(trim, "### "):
			b.WriteString(indent + styleAmber.Render(trim) + "\n")
		case strings.HasPrefix(trim, "---"):
			b.WriteString(indent + styleDim.Render(strings.Repeat("─", 40)) + "\n")
		case strings.HasPrefix(trim, "- "), strings.HasPrefix(trim, "• "):
			b.WriteString(indent + "  " + styleText.Render(trim) + "\n")
		case strings.HasPrefix(trim, "**") && strings.Contains(trim, "**"):
			b.WriteString(indent + styleGold.Render(trim) + "\n")
		default:
			b.WriteString(indent + styleText.Render(trim) + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}
