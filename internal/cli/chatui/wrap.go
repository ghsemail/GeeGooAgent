package chatui

import (
	"strings"
	"unicode"

	"github.com/mattn/go-runewidth"
)

// ContentWrapWidth is the usable text column inside the TUI viewport.
func ContentWrapWidth(terminalWidth int) int {
	w := terminalWidth - 4
	if w < 32 {
		return 32
	}
	if w > 96 {
		return 96
	}
	return w
}

// PanelContentWidth is the inner text width inside a process/clarify panel.
func PanelContentWidth(terminalWidth int) int {
	w := clampRuleWidth(terminalWidth) - 4
	if w < 24 {
		return 24
	}
	return w
}

// WrapPlain hard-wraps plain text to width (display columns), preserving explicit newlines.
func WrapPlain(text string, width int) string {
	if width < 1 || text == "" {
		return text
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	var out []string
	for _, para := range strings.Split(text, "\n") {
		para = strings.TrimRight(para, " \t")
		if strings.TrimSpace(para) == "" {
			out = append(out, "")
			continue
		}
		out = append(out, wrapParagraph(para, width)...)
	}
	return strings.Join(out, "\n")
}

// WrapWithPrefix wraps text with a first-line prefix and continuation indent.
func WrapWithPrefix(text, prefix, indent string, width int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return prefix
	}
	prefixW := runewidth.StringWidth(prefix)
	if indent == "" {
		indent = strings.Repeat(" ", prefixW)
	}
	indentW := runewidth.StringWidth(indent)

	var lines []string
	remaining := text
	first := true
	for remaining != "" {
		lineW := width - prefixW
		pfx := prefix
		if !first {
			lineW = width - indentW
			pfx = indent
		}
		if lineW < 8 {
			lineW = 8
		}
		chunk, rest := takeWrappedLine(remaining, lineW)
		lines = append(lines, pfx+chunk)
		remaining = strings.TrimLeft(rest, " \t")
		first = false
	}
	return strings.Join(lines, "\n")
}

func takeWrappedLine(s string, width int) (line, rest string) {
	if width < 1 {
		return s, ""
	}
	if runewidth.StringWidth(s) <= width {
		return s, ""
	}
	parts := wrapParagraph(s, width)
	if len(parts) == 0 {
		return "", s
	}
	line = parts[0]
	rest = strings.TrimPrefix(s, line)
	return line, rest
}

func wrapParagraph(s string, width int) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if runewidth.StringWidth(s) <= width {
		return []string{s}
	}

	var lines []string
	runes := []rune(s)
	start := 0
	for start < len(runes) {
		end, cut := nextWrapCut(runes[start:], width)
		if end == 0 {
			end = 1
			cut = 1
		}
		line := strings.TrimRight(string(runes[start:start+cut]), " \t\u3000")
		if line != "" {
			lines = append(lines, line)
		}
		start += end
		for start < len(runes) && unicode.IsSpace(runes[start]) {
			start++
		}
	}
	return lines
}

func nextWrapCut(runes []rune, width int) (consumed, breakAt int) {
	if len(runes) == 0 {
		return 0, 0
	}
	w := 0
	lastBreak := 0
	i := 0
	for i < len(runes) {
		rw := runewidth.RuneWidth(runes[i])
		if rw <= 0 {
			i++
			continue
		}
		if w+rw > width {
			break
		}
		w += rw
		i++
		if isWrapBreakAfter(runes[i-1]) || unicode.IsSpace(runes[i-1]) {
			lastBreak = i
		}
	}
	if i == 0 {
		return 1, 1
	}
	if lastBreak > 0 {
		return i, lastBreak
	}
	return i, i
}

func isWrapBreakAfter(r rune) bool {
	switch r {
	case ' ', '\u3000', ',', '.', ';', ':', '!', '?',
		'，', '。', '；', '！', '？', '、', '：',
		')', '）', ']', '】', '」', '》', '"', '\u201d', '\'':
		return true
	default:
		return false
	}
}

func leadingSpaceWidth(s string) int {
	n := 0
	for _, r := range s {
		if r == ' ' {
			n++
		} else {
			break
		}
	}
	return n
}
