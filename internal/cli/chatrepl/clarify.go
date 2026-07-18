package chatrepl

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func (r *Repl) attachClarify() {
	// default handler; TUI may override via SetClarifyFn
}

func (r *Repl) SetClarifyFn(fn tools.ClarifyFunc) {
	r.clarifyFn = fn
}

func (r *Repl) promptClarify(question string, choices []string) (string, bool) {
	if r.clarifyFn != nil {
		return r.clarifyFn(question, choices)
	}
	return promptClarifyCLI(r, question, choices)
}

func promptClarifyCLI(r *Repl, question string, choices []string) (string, bool) {
	if r == nil || r.UI == nil {
		return "", false
	}
	r.UI.PrintInfo("? " + question)
	options := tools.ClarifyDisplayOptions(choices)
	if len(options) == 0 {
		r.UI.PrintPrompt()
		line, err := readClarifyLine(r)
		if err != nil {
			return "", false
		}
		line = strings.TrimSpace(line)
		return line, line != ""
	}
	for i, opt := range options {
		r.UI.PrintInfo(fmt.Sprintf("  [%s] %s", tools.ClarifyChoiceLabel(i), opt))
	}
	r.UI.PrintInfo("输入 A/B/C…、编号、选项全文，或 Esc 跳过（空行跳过）")
	r.UI.PrintPrompt()
	line, err := readClarifyLine(r)
	if err != nil {
		return "", false
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return "", true
	}
	if ans, ok := matchClarifyInput(line, options); ok {
		if len(choices) > 0 && ans == tools.ClarifyOtherLabel {
			r.UI.PrintPrompt()
			other, err := readClarifyLine(r)
			if err != nil {
				return "", false
			}
			other = strings.TrimSpace(other)
			return other, other != ""
		}
		return ans, true
	}
	return line, true
}

func readClarifyLine(r *Repl) (string, error) {
	in := r.stdin
	if in == nil {
		in = strings.NewReader("")
	}
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func matchClarifyInput(line string, options []string) (string, bool) {
	upper := strings.ToUpper(strings.TrimSpace(line))
	if len(upper) == 1 && upper[0] >= 'A' && int(upper[0]-'A') < len(options) {
		return options[int(upper[0]-'A')], true
	}
	if n, err := parseClarifyIndex(line, len(options)); err == nil {
		return options[n], true
	}
	for _, opt := range options {
		if strings.EqualFold(strings.TrimSpace(opt), line) {
			return opt, true
		}
	}
	return "", false
}

func parseClarifyIndex(line string, n int) (int, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return 0, fmt.Errorf("empty")
	}
	var idx int
	if _, err := fmt.Sscanf(line, "%d", &idx); err != nil {
		return 0, err
	}
	if idx >= 1 && idx <= n {
		return idx - 1, nil
	}
	return 0, fmt.Errorf("out of range")
}
