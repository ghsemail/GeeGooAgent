package chattui

import (
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

type clarifyAnswer struct {
	Answer string
	OK     bool
}

type clarifyAsk struct {
	Question string
	Choices  []string
}

func (m *Model) clarifyDisplayOptions() []string {
	return tools.ClarifyDisplayOptions(m.clarifyChoices)
}

func (m *Model) clearClarify() {
	m.clarifyPending = false
	m.clarifyAwaitingText = false
	m.clarifyQuestion = ""
	m.clarifyChoices = nil
	m.clarifyFocus = 0
}

func (m *Model) submitClarifyAnswer(answer string, ok bool) {
	if h := m.activeHost(); h != nil {
		h.AnswerClarify(answer, ok)
	}
	m.clearClarify()
	m.info = ""
}

func (m *Model) answerClarifyChoice(idx int) {
	opts := m.clarifyDisplayOptions()
	if idx < 0 || idx >= len(opts) {
		return
	}
	if len(m.clarifyChoices) > 0 && idx == len(opts)-1 {
		m.clarifyPending = false
		m.clarifyAwaitingText = true
		m.input.SetValue("")
		m.info = "请输入你的回答"
		return
	}
	m.submitClarifyAnswer(opts[idx], true)
}

func (m *Model) handleClarifyKey(msg tea.KeyMsg) bool {
	if !m.clarifyPending {
		return false
	}
	opts := m.clarifyDisplayOptions()
	switch msg.Type {
	case tea.KeyUp:
		if m.clarifyFocus > 0 {
			m.clarifyFocus--
		}
		return true
	case tea.KeyDown:
		if m.clarifyFocus < len(opts)-1 {
			m.clarifyFocus++
		}
		return true
	case tea.KeyEnter:
		m.answerClarifyChoice(m.clarifyFocus)
		return true
	case tea.KeyEsc:
		m.submitClarifyAnswer("", true)
		return true
	}
	if len(msg.Runes) == 1 {
		r := msg.Runes[0]
		if r >= 'a' && r <= 'z' {
			r = unicode.ToUpper(r)
		}
		if r >= 'A' && r <= 'Z' {
			idx := int(r - 'A')
			if idx < len(opts) {
				m.answerClarifyChoice(idx)
				return true
			}
		}
		if r >= '1' && r <= '9' {
			idx := int(r - '1')
			if idx < len(opts) {
				m.answerClarifyChoice(idx)
				return true
			}
		}
	}
	if key := strings.ToLower(msg.String()); len(key) == 1 && key[0] >= 'a' && key[0] <= 'z' {
		idx := int(key[0] - 'a')
		if idx < len(opts) {
			m.answerClarifyChoice(idx)
			return true
		}
	}
	return true
}
