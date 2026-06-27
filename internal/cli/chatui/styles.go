package chatui

import "github.com/charmbracelet/lipgloss"

var (
	styleGold   = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGold))
	styleAmber  = lipgloss.NewStyle().Foreground(lipgloss.Color(colorAmber))
	styleDim    = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim))
	styleErr    = lipgloss.NewStyle().Foreground(lipgloss.Color(colorErr))
	styleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#B8860B")).
			Padding(1, 2)
)
