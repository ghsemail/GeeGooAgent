package chatui

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reGlueHeading = regexp.MustCompile(`([^\n])(#{2,6}\s)`)
	reGlueH3Num   = regexp.MustCompile(`([^\n#])(#{3,6}\s*\d+\.\s)`)
	reGlueHeadingCN = regexp.MustCompile(`([！。!?；:，,])(#{2,6})`)
	reGlueHeadingMid = regexp.MustCompile(`([^\n#])(#{2,6}\d+\.)`)
	reGlueHeadingTight = regexp.MustCompile(`([^\n#\s])(#{2,6}\d)`)
	reHeadingSpace = regexp.MustCompile(`^(#{1,6})([^\s#\n])`)
	reHeadingNumSpace = regexp.MustCompile(`^(#{1,6})(\d+\.)`)
	reGlueCard    = regexp.MustCompile(`([^\n])(\*\*[0-9]+\.)`)
	reGlueSummary = regexp.MustCompile(`([^\n])(小结[:：])`)
	reGlueHR      = regexp.MustCompile(`\s*---+\s*`)
	rePipeField   = regexp.MustCompile(`\s\|\s*((?:ID|Bot ID|bot_id|状态|网格区间|网格范围|档数|当前档|持仓|成本价|当前价|浮盈亏|盈亏|操作|触发)[：:])`)
	reGlueField     = regexp.MustCompile(`\s+-\s+((?:代码|类型|状态|网格|持仓|盈亏|频率|成本|当前|对冲)[：:])`)
	reGlueNumbered  = regexp.MustCompile(`([^\n\d])(\d+\.\s)`)
	reGlueListDash  = regexp.MustCompile(`([^\n-])(\s+-\s+)`)
	reGlueColonList   = regexp.MustCompile(`([：:;；])\s*-\s+`)
	reGluePunctList   = regexp.MustCompile(`([。！？])\s*-\s+`)
	rePipeHRHeading   = regexp.MustCompile(`\|+---+\s*(#{1,6})`)
	reHRHeading       = regexp.MustCompile(`---+(\s*#{1,6})`)
	reCNHeadingSpace  = regexp.MustCompile(`(#{2,6})([\p{Han}一二三四五六七八九十、（(])`)
	reHeadingTableGlue = regexp.MustCompile(`([面论表况])\|([^\n|]+\|[^\n|]+\|)`)
	reGlueBlockquote  = regexp.MustCompile(`([）)％%港元\d\.])(>[^|\n#]+)`)
	rePipeBlockquote    = regexp.MustCompile(`\|(\s*>[^|\n#]+)`)
	reTitleBlockquote   = regexp.MustCompile(`(综合分析|结论|建议)(>[^|\n#]+)`)
	reAdviceList      = regexp.MustCompile(`(操作建议[：:])\s*-`)
	reAdviceDash      = regexp.MustCompile(`([者。盈%）)])\s*-([\p{Han}])`)
	reFixTightDashList = regexp.MustCompile(`\n-([\p{Han}A-Za-z])`)
	reDateCell        = regexp.MustCompile(`^\d{1,2}/\d{1,2}$`)
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
			if strings.Contains(trim, "#") {
				for _, sub := range splitPipeHeadingGlue(line) {
					out = append(out, sub)
				}
				continue
			}
			out = append(out, line)
			continue
		}
		line = strings.ReplaceAll(line, " ### ", "\n### ")
		line = reGlueHeadingCN.ReplaceAllString(line, "$1\n$2")
		line = reGlueHeadingMid.ReplaceAllString(line, "$1\n$2")
		line = reGlueHeadingTight.ReplaceAllString(line, "$1\n$2")
		line = reGlueHeading.ReplaceAllString(line, "$1\n$2")
		line = reGlueH3Num.ReplaceAllString(line, "$1\n$2")
		line = reGlueCard.ReplaceAllString(line, "$1\n$2")
		line = reGlueHR.ReplaceAllString(line, "\n")
		line = reGlueSummary.ReplaceAllString(line, "$1\n$2")
		line = rePipeField.ReplaceAllString(line, "\n  $1")
		line = reGlueField.ReplaceAllString(line, "\n  - $1")
		line = reGlueNumbered.ReplaceAllString(line, "$1\n$2")
		line = reGlueColonList.ReplaceAllString(line, "$1\n- ")
		line = reGluePunctList.ReplaceAllString(line, "$1\n- ")
		line = reGlueListDash.ReplaceAllString(line, "$1\n- ")
		for _, sub := range strings.Split(line, "\n") {
			sub = strings.TrimRight(sub, " ")
			sub = fixHeadingSyntax(sub)
			if strings.TrimSpace(sub) == "" {
				continue
			}
			out = append(out, sub)
		}
	}
	return breakInlinePipeFields(strings.Join(out, "\n"))
}

func fixHeadingSyntax(line string) string {
	line = reHeadingSpace.ReplaceAllString(line, "$1 $2")
	line = reHeadingNumSpace.ReplaceAllString(line, "$1 $2")
	return line
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
	text = normalizeGluedAnalysisMarkdown(text)
	text = normalizeGluedMarkdownTables(text)
	text = NormalizeAssistantLayout(text)
	text = ensureListSpacing(text)
	if strings.Contains(text, "|") {
		text = convertMarkdownTables(text)
	}
	return tightenParagraphSpacing(text)
}

// normalizeGluedAnalysisMarkdown fixes stock-analysis replies glued with |---, ---###,
// inline tables, and blockquotes (common when models stream without line breaks).
func normalizeGluedAnalysisMarkdown(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = rePipeHRHeading.ReplaceAllString(text, "\n$1")
	text = reHRHeading.ReplaceAllString(text, "\n$1")
	text = reCNHeadingSpace.ReplaceAllString(text, "$1 $2")
	text = reHeadingTableGlue.ReplaceAllString(text, "$1\n\n|$2")
	text = reGlueBlockquote.ReplaceAllString(text, "$1\n$2")
	text = rePipeBlockquote.ReplaceAllString(text, "\n$1")
	text = reTitleBlockquote.ReplaceAllString(text, "$1\n$2")
	text = reAdviceList.ReplaceAllString(text, "$1\n-")
	text = reAdviceDash.ReplaceAllString(text, "$1\n-$2")
	text = reFixTightDashList.ReplaceAllString(text, "\n- $1")
	text = strings.ReplaceAll(text, "|  |", "\n|")
	return strings.TrimSpace(text)
}

func splitPipeHeadingGlue(line string) []string {
	trim := strings.TrimSpace(line)
	trim = strings.TrimLeft(trim, "|")
	trim = strings.TrimLeft(trim, "-")
	trim = strings.TrimSpace(trim)
	if trim == "" {
		return nil
	}
	var out []string
	for strings.Contains(trim, "##") {
		idx := strings.Index(trim, "##")
		if idx > 0 {
			prefix := strings.TrimSpace(trim[:idx])
			prefix = strings.Trim(prefix, "|- ")
			if prefix != "" && !isTableSeparator(prefix) {
				out = append(out, prefix)
			}
		}
		trim = strings.TrimSpace(trim[idx:])
		end := len(trim)
		if j := strings.Index(trim, "|"); j > 0 && strings.Count(trim[j:], "|") >= 2 {
			heading := strings.TrimSpace(trim[:j])
			heading = strings.TrimRight(heading, "-")
			out = append(out, heading)
			trim = strings.TrimSpace(trim[j:])
			continue
		}
		out = append(out, trim[:end])
		break
	}
	if len(out) == 0 {
		return []string{line}
	}
	if strings.TrimSpace(trim) != "" && !strings.HasPrefix(trim, "##") {
		out = append(out, trim)
	}
	return out
}

// normalizeGluedMarkdownTables splits model-glued table rows (||) and inline headers
// such as "## 标题|#|名称|代码||1|foo|..." before card conversion.
func normalizeGluedMarkdownTables(text string) string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		out = append(out, expandGluedTableLine(line)...)
	}
	return strings.Join(out, "\n")
}

func expandGluedTableLine(line string) []string {
	trim := strings.TrimSpace(line)
	if trim == "" {
		return []string{line}
	}
	if !looksLikeGluedTableLine(trim) {
		return []string{line}
	}
	if idx := findInlineTableHeaderStart(trim); idx > 0 {
		prefix := strings.TrimSpace(trim[:idx])
		suffix := strings.TrimSpace(trim[idx:])
		var rows []string
		if prefix != "" {
			rows = append(rows, prefix)
		}
		rows = append(rows, expandGluedPipeRows(suffix)...)
		return rows
	}
	if strings.Contains(trim, "||") {
		return expandGluedPipeRows(trim)
	}
	return []string{line}
}

func looksLikeGluedTableLine(line string) bool {
	if strings.Contains(line, "||") && strings.Count(line, "|") >= 3 {
		return true
	}
	return findInlineTableHeaderStart(line) > 0
}

func findInlineTableHeaderStart(line string) int {
	best := -1
	for _, marker := range []string{"|#|", "| # |", "|序号|"} {
		if i := strings.Index(line, marker); i > 0 {
			if best < 0 || i < best {
				best = i
			}
		}
	}
	return best
}

func expandGluedPipeRows(s string) []string {
	parts := strings.Split(s, "||")
	var out []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "#") {
			out = append(out, part)
			continue
		}
		if strings.Count(part, "|") < 2 {
			out = append(out, part)
			continue
		}
		out = append(out, normalizeTableRowLine(part))
	}
	return out
}

func normalizeTableRowLine(part string) string {
	part = strings.TrimSpace(part)
	if !strings.HasPrefix(part, "|") {
		part = "|" + part
	}
	if !strings.HasSuffix(part, "|") {
		part += "|"
	}
	return part
}

func ensureListSpacing(text string) string {
	lines := strings.Split(text, "\n")
	var out []string
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if isListLine(trim) && i > 0 {
			prev := strings.TrimSpace(lines[i-1])
			if prev != "" && !isListLine(prev) && (len(out) == 0 || out[len(out)-1] != "") {
				out = append(out, "")
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func isListLine(line string) bool {
	return strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || strings.HasPrefix(line, "+ ")
}

func convertMarkdownTables(text string) string {
	lines := strings.Split(text, "\n")
	var out []string
	for i := 0; i < len(lines); {
		cells := parseTableRow(lines[i])
		if cells != nil && !isTableSeparator(lines[i]) {
			if len(cells) == 2 && looksLikeDateCell(cells[0]) {
				if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
					out = append(out, "")
				}
				out = append(out, formatKeyValueTable([][]string{cells}))
				i++
				continue
			}
			tableLines := []string{lines[i]}
			i++
			for i < len(lines) {
				if isTableSeparator(lines[i]) {
					tableLines = append(tableLines, lines[i])
					i++
					continue
				}
				next := parseTableRow(lines[i])
				if next == nil {
					break
				}
				if len(tableLines) > 0 && isLikelyTableHeader(next) && tableHasDataRows(tableLines) {
					break
				}
				tableLines = append(tableLines, lines[i])
				i++
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

func tableHasDataRows(tableLines []string) bool {
	start := 1
	if len(tableLines) > 1 && isTableSeparator(tableLines[1]) {
		start = 2
	}
	return len(tableLines) > start
}

func tightenParagraphSpacing(text string) string {
	lines := strings.Split(text, "\n")
	var out []string
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "#" || trim == "##" {
			continue
		}
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
	trim := strings.TrimSpace(line)
	return strings.HasPrefix(trim, "##") ||
		strings.HasPrefix(trim, "###") ||
		strings.HasPrefix(trim, "**") ||
		strings.HasPrefix(trim, ">") ||
		strings.HasPrefix(trim, "小结")
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
	if len(tableLines) == 0 {
		return ""
	}
	first := parseTableRow(tableLines[0])
	if first == nil {
		return ""
	}
	headers := first
	start := 1
	if isLikelyTableHeader(first) {
		if start < len(tableLines) && isTableSeparator(tableLines[start]) {
			start++
		}
	} else {
		headers = defaultTableHeaders(len(first))
		start = 0
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
	if len(headers) == 2 && isKeyValueTableHeaders(headers) {
		return formatKeyValueTable(rows)
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

func isLikelyTableHeader(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	if looksLikeDateCell(cells[0]) {
		return false
	}
	h := strings.TrimSpace(strings.ToLower(cells[0]))
	switch h {
	case "#", "序号", "no", "no.":
		return true
	}
	if strings.Contains(cells[0], "名称") || strings.Contains(strings.ToLower(cells[0]), "name") {
		return true
	}
	if len(cells) >= 2 && !looksLikeColumnName(cells[1]) {
		return false
	}
	return !isNumericIndex(cells[0])
}

func looksLikeColumnName(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || len([]rune(s)) > 10 {
		return false
	}
	for _, w := range []string{"积极", "出货", "迹象", "港元", "流入", "升高", "增大", "调整"} {
		if strings.Contains(s, w) {
			return false
		}
	}
	return true
}

func looksLikeDateCell(s string) bool {
	return reDateCell.MatchString(strings.TrimSpace(s))
}

func isKeyValueTableHeaders(headers []string) bool {
	if len(headers) != 2 {
		return false
	}
	a := strings.TrimSpace(headers[0])
	b := strings.TrimSpace(headers[1])
	switch {
	case strings.Contains(a, "日期") || strings.Contains(b, "事件"):
		return true
	case strings.Contains(a, "维度") || strings.Contains(b, "信号"):
		return true
	case strings.Contains(a, "类型") || strings.Contains(b, "净流入"):
		return true
	}
	return len([]rune(a)) <= 8 && len([]rune(b)) <= 12 &&
		!strings.Contains(a, "名称") && !strings.Contains(a, "#")
}

func formatKeyValueTable(rows [][]string) string {
	var b strings.Builder
	n := 0
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		if isSeparatorCells(row) {
			continue
		}
		k := strings.TrimSpace(row[0])
		v := strings.TrimSpace(row[1])
		if k == "" && v == "" {
			continue
		}
		if n > 0 {
			b.WriteByte('\n')
		}
		n++
		if k == "" {
			b.WriteString("- ")
			b.WriteString(v)
			continue
		}
		b.WriteString("- **")
		b.WriteString(k)
		b.WriteString("**：")
		b.WriteString(v)
	}
	return strings.TrimRight(b.String(), "\n")
}

func isSeparatorCells(row []string) bool {
	line := "|" + strings.Join(row, "|") + "|"
	return isTableSeparator(line)
}

func defaultTableHeaders(n int) []string {
	defaults := []string{"#", "名称", "代码", "频率", "买入信号", "状态", "网格区间", "档数", "盈亏"}
	if n <= len(defaults) {
		return defaults[:n]
	}
	out := make([]string, n)
	for i := range out {
		if i < len(defaults) {
			out[i] = defaults[i]
		} else {
			out[i] = fmt.Sprintf("字段%d", i+1)
		}
	}
	return out
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

