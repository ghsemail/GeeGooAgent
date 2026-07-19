package chattui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatui"
)

func (m Model) slashMatches() []chatui.SlashCommand {
	return chatui.MatchSlashCommands(m.input.Value())
}

func (m Model) slashMenuOpen() bool {
	if m.sessionPicker || m.approvalPending || m.activeSlotPlanPending() {
		return false
	}
	val := strings.TrimSpace(m.input.Value())
	return strings.HasPrefix(val, "/") && len(m.slashMatches()) > 0
}

func (m *Model) clampSlashPick() {
	matches := m.slashMatches()
	if len(matches) == 0 {
		m.slashPick = 0
		return
	}
	if m.slashPick < 0 {
		m.slashPick = 0
	}
	if m.slashPick >= len(matches) {
		m.slashPick = len(matches) - 1
	}
}

func (m Model) acceptSlashSuggestion() (Model, tea.Cmd) {
	matches := m.slashMatches()
	if len(matches) == 0 {
		return m, nil
	}
	pick := m.slashPick
	if pick < 0 || pick >= len(matches) {
		pick = 0
	}
	m.input.SetValue(matches[pick].Command + " ")
	m.slashPick = 0
	return m, nil
}

func renderSlashMenu(matches []chatui.SlashCommand, pick, width int) string {
	if len(matches) == 0 {
		return ""
	}
	maxRows := 8
	if len(matches) < maxRows {
		maxRows = len(matches)
	}
	start := 0
	if pick >= maxRows {
		start = pick - maxRows + 1
	}
	end := start + maxRows
	if end > len(matches) {
		end = len(matches)
	}

	descW := 40
	if width > 20 {
		descW = width - 22
		if descW > 56 {
			descW = 56
		}
		if descW < 16 {
			descW = 16
		}
	}

	var b strings.Builder
	b.WriteString(styleDim.Render("命令"))
	b.WriteByte('\n')
	for i := start; i < end; i++ {
		item := matches[i]
		cmd := item.Command
		desc := item.Description
		if len(desc) > descW {
			desc = TruncateRunes(desc, descW)
		}
		pad := 18 - len(cmd)
		if pad < 1 {
			pad = 1
		}
		line := cmd + strings.Repeat(" ", pad) + desc
		if i == pick {
			b.WriteString(styleFocus.Render("› " + line))
		} else {
			b.WriteString(styleDim.Render("  " + line))
		}
		b.WriteByte('\n')
	}
	if len(matches) > maxRows {
		b.WriteString(styleDim.Render(fmt.Sprintf("  … 还有 %d 条", len(matches)-maxRows)))
	}
	return strings.TrimRight(b.String(), "\n")
}
