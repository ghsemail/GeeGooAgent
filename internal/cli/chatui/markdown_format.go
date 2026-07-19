package chatui

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reGlueHeading = regexp.MustCompile(`([^\n])(#{2,6}\s)`)
	reGlueH3Num   = regexp.MustCompile(`([^\n#])(#{3,6}\s*\d+\.\s)`)
	reGlueCard    = regexp.MustCompile(`([^\n])(\*\*[0-9]+\.)`)
	reGlueSummary = regexp.MustCompile(`([^\n])(小结[:：])`)
	reGlueHR      = regexp.MustCompile(`\s*---+\s*`)
	rePipeField   = regexp.MustCompile(`\s\|\s*((?:ID|Bot ID|bot_id|状态|网格区间|网格范围|档数|当前档|持仓|成本价|当前价|浮盈亏|盈亏|操作|触发)[：:])`)
	reGlueField     = regexp.MustCompile(`\s+-\s+((?:代码|类型|状态|网格|持仓|盈亏|频率|成本|当前|对冲)[：:])`)
	reGlueNumbered  = regexp.MustCompile(`([^\n\d])(\d+\.\s)`)
	reGlueListDash  = regexp.MustCompile(`([^\n-])(\s+-\s+)`)
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
			out = append(out, "")
			continue
		}
		if strings.HasPrefix(trim, "|") || isTableSeparator(trim) {
			out = append(out, line)
			continue
		}
		line = strings.ReplaceAll(line, " ### ", "\n### ")
		line = reGlueHeading.ReplaceAllString(line, "$1\n$2")
		line = reGlueH3Num.ReplaceAllString(line, "$1\n$2")
		line = reGlueCard.ReplaceAllString(line, "$1\n$2")
		line = reGlueHR.ReplaceAllString(line, "\n")
		line = reGlueSummary.ReplaceAllString(line, "$1\n$2")
		line = rePipeField.ReplaceAllString(line, "\n  $1")
		line = reGlueField.ReplaceAllString(line, "\n  - $1")
		line = reGlueNumbered.ReplaceAllString(line, "$1\n$2")
		line = reGlueListDash.ReplaceAllString(line, "$1\n$2")
		for _, sub := range strings.Split(line, "\n") {
			sub = strings.TrimRight(sub, " ")
			if strings.TrimSpace(sub) == "" {
				continue
			}
			out = append(out, sub)
		}
	}
	return breakInlinePipeFields(strings.Join(out, "\n"))
}

func breakAfterPunctuation(line string) string {
	runes := []rune(line)
	if len(runes) == 0 {
		return line
	}
	var b strings.Builder
	for i, r := range runes {
		b.WriteRune(r)
		if (r == '。' || r == '！' || r == '？') && i+1 < len(runes) && runes[i+1] != '\n' {
			b.WriteByte('\n')
		}
	}
	return b.String()
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
					out = append(out, "  "+p)
				}
			}
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// PreprocessTerminalMarkdown adapts assistant markdown for narrow terminals.
func PreprocessTerminalMarkdown(text string) string {
	text = NormalizeAssistantLayout(text)
	if strings.Contains(text, "|") {
		text = convertMarkdownTables(text)
	}
	return tightenParagraphSpacing(text)
}

func convertMarkdownTables(text string) string {
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

func tightenParagraphSpacing(text string) string {
	lines := strings.Split(text, "\n")
	var out []string
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || trim == "---" || trim == "***" || strings.Trim(trim, "-*") == "" {
			if len(out) > 0 && out[len(out)-1] != "" {
				out = append(out, "")
			}
			continue
		}
		if isSectionStart(trim) && len(out) > 0 && out[len(out)-1] != "" {
			out = append(out, "")
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func isSectionStart(line string) bool {
	return strings.HasPrefix(line, "##") ||
		strings.HasPrefix(line, "**") ||
		strings.HasPrefix(line, "小结")
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
	var fields []string

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
		case strings.Contains(h, "名称") || hl == "name" || hl == "botname":
			title = v
			continue
		case strings.Contains(h, "代码") || hl == "code":
			code = v
			continue
		}
		fields = append(fields, h+"："+v)
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
	for _, f := range fields {
		b.WriteByte('\n')
		b.WriteString("  ")
		b.WriteString(f)
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

// RenderPlainAssistantBody renders assistant text line-by-line for the terminal.
// wrapW is the max content width in runes (viewport does not soft-wrap).
func RenderPlainAssistantBody(text string, wrapW int) string {
	text = PreprocessTerminalMarkdown(text)
	if strings.TrimSpace(text) == "" {
		return styleDim.Render("⋯ 正在生成回复…")
	}
	if wrapW < 32 {
		wrapW = 32
	}
	var b strings.Builder
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, " ")
		if line == "" {
			b.WriteByte('\n')
			continue
		}
		trim := strings.TrimLeft(line, " ")
		indent := leadingSpaceWidth(line)
		lineW := wrapW - indent
		if lineW < 24 {
			lineW = 24
		}
		plain, style := classifyAssistantLine(trim)
		plain = stripInlineMarkdown(plain)
		indentStr := strings.Repeat(" ", indent)
		for _, wl := range strings.Split(WrapPlain(plain, lineW), "\n") {
			if wl == "" {
				b.WriteByte('\n')
				continue
			}
			b.WriteString(style(indentStr + wl))
			b.WriteByte('\n')
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func classifyAssistantLine(trim string) (plain string, style func(string) string) {
	isHeading := strings.HasPrefix(strings.TrimSpace(trim), "#")
	plain = cleanMarkdownHeadingPrefix(trim)
	switch {
	case strings.HasPrefix(plain, "- "), strings.HasPrefix(plain, "• "):
		body := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(plain, "• "), "- "))
		return body, func(s string) string { return styleText.Render(s) }
	case strings.HasPrefix(plain, "**") && strings.Contains(plain, "**"):
		return plain, func(s string) string { return styleGold.Render(s) }
	case isHeading || looksNumberedSection(plain):
		if looksNumberedSection(plain) {
			return plain, func(s string) string { return styleAmber.Render(s) }
		}
		return plain, func(s string) string { return styleGold.Render(s) }
	default:
		return plain, func(s string) string { return styleText.Render(s) }
	}
}

func cleanMarkdownHeadingPrefix(s string) string {
	s = strings.TrimSpace(s)
	for strings.HasPrefix(s, "#") {
		i := 0
		for i < len(s) && s[i] == '#' {
			i++
		}
		s = strings.TrimSpace(s[i:])
	}
	return s
}

func looksNumberedSection(s string) bool {
	s = strings.TrimSpace(s)
	dot := strings.IndexByte(s, '.')
	if dot <= 0 || dot > 3 {
		return false
	}
	for _, r := range s[:dot] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return dot+1 < len(s)
}

// stripInlineMarkdown removes lightweight markdown markers the plain renderer does not parse.
func stripInlineMarkdown(s string) string {
	for {
		i := strings.Index(s, "`")
		if i < 0 {
			break
		}
		rest := s[i+1:]
		j := strings.Index(rest, "`")
		if j < 0 {
			break
		}
		s = s[:i] + rest[:j] + rest[j+1:]
	}
	for {
		i := strings.Index(s, "**")
		if i < 0 {
			break
		}
		rest := s[i+2:]
		j := strings.Index(rest, "**")
		if j < 0 {
			break
		}
		s = s[:i] + rest[:j] + rest[j+2:]
	}
	return s
}

