package llm

import (
	"regexp"
	"strings"
)

const thinkTagPattern = `(?:redacted_thinking|think)`

var (
	inlineThinkBlockRE    = regexp.MustCompile(`(?is)<` + thinkTagPattern + `>(.*?)</` + thinkTagPattern + `>`)
	inlineUnclosedThinkRE = regexp.MustCompile(`(?is)<` + thinkTagPattern + `>[\s\S]*$`)
)

// SplitInlineThinking separates visible answer text from inline thinking blocks in content.
func SplitInlineThinking(s string) (visible, extracted string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	var blocks []string
	for _, m := range inlineThinkBlockRE.FindAllStringSubmatch(s, -1) {
		if len(m) > 1 {
			if body := strings.TrimSpace(m[1]); body != "" {
				blocks = append(blocks, body)
			}
		}
	}
	visible = strings.TrimSpace(inlineThinkBlockRE.ReplaceAllString(s, ""))
	visible = strings.TrimSpace(inlineUnclosedThinkRE.ReplaceAllString(visible, ""))
	return visible, strings.Join(blocks, "\n")
}

// AnswerStreamAccumulator strips inline thinking from streamed content deltas and emits only new visible text.
type AnswerStreamAccumulator struct {
	raw           strings.Builder
	visiblePrev   string
	thinkingPrev  string
	OnVisible     func(chunk string)
	OnThinking    func(chunk string)
}

func (a *AnswerStreamAccumulator) PushContent(delta string) {
	if a == nil || delta == "" {
		return
	}
	a.raw.WriteString(delta)
	visible, thinking := SplitInlineThinking(a.raw.String())
	if thinking != a.thinkingPrev {
		if add := strings.TrimPrefix(thinking, a.thinkingPrev); add != "" && a.OnThinking != nil {
			a.OnThinking(add)
		}
		a.thinkingPrev = thinking
	}
	if len(visible) > len(a.visiblePrev) && strings.HasPrefix(visible, a.visiblePrev) {
		chunk := visible[len(a.visiblePrev):]
		a.visiblePrev = visible
		if chunk != "" && a.OnVisible != nil {
			a.OnVisible(chunk)
		}
	} else if visible != a.visiblePrev {
		// Model rewrote prefix; emit full visible once.
		a.visiblePrev = visible
		if visible != "" && a.OnVisible != nil {
			a.OnVisible(visible)
		}
	}
}

func (a *AnswerStreamAccumulator) Visible() string {
	v, _ := SplitInlineThinking(a.raw.String())
	return v
}

func (a *AnswerStreamAccumulator) ThinkingExtracted() string {
	_, t := SplitInlineThinking(a.raw.String())
	return t
}
