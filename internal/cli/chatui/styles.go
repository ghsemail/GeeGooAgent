package chatui

import "github.com/charmbracelet/lipgloss"

var (
	styleGold    = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGold)).Bold(true)
	styleAmber   = lipgloss.NewStyle().Foreground(lipgloss.Color(colorAmber)).Bold(true)
	styleAccent  = lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent))
	styleText      = lipgloss.NewStyle().Foreground(lipgloss.Color(colorText))
	styleThinking  = lipgloss.NewStyle().Foreground(lipgloss.Color(colorThinking))
	styleDim       = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim))
	styleErr     = lipgloss.NewStyle().Foreground(lipgloss.Color(colorErr)).Bold(true)
	styleOK      = lipgloss.NewStyle().Foreground(lipgloss.Color(colorOK))
	stylePanel   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorBorder)).
			Padding(0, 2)
	styleSubtle  = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim)).Italic(true)
	styleToolRun = lipgloss.NewStyle().Foreground(lipgloss.Color(colorText))
)
