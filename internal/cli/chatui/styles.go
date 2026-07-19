package chatui

import "github.com/charmbracelet/lipgloss"

// Semantic text styles (typography hierarchy).
var (
	styleBrand   = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGold)).Bold(true)
	styleTitle   = lipgloss.NewStyle().Foreground(lipgloss.Color(colorAmber)).Bold(true)
	styleSection = lipgloss.NewStyle().Foreground(lipgloss.Color(colorText)).Bold(true)
	styleBody    = lipgloss.NewStyle().Foreground(lipgloss.Color(colorText))
	styleUser    = lipgloss.NewStyle().Foreground(lipgloss.Color(colorUser))
	styleMeta    = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim))
	styleWhisper = lipgloss.NewStyle().Foreground(lipgloss.Color(colorWhisper))
	styleThinking = lipgloss.NewStyle().Foreground(lipgloss.Color(colorThinking)).Italic(true)
	styleSubtle  = styleThinking
	styleTool    = lipgloss.NewStyle().Foreground(lipgloss.Color(colorUser))

	styleGold    = styleBrand
	styleAmber   = styleTitle
	styleAccent  = lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent))
	styleText    = styleBody
	styleDim     = styleMeta
	styleErr     = lipgloss.NewStyle().Foreground(lipgloss.Color(colorErr)).Bold(true)
	styleOK      = lipgloss.NewStyle().Foreground(lipgloss.Color(colorOK))
	styleRunning = lipgloss.NewStyle().Foreground(lipgloss.Color(colorRunning))

	stylePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorBorder)).
			Padding(0, 2)
	styleProcessBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorWhisper)).
			Padding(0, 1)
	styleInputBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorBorder)).
			Padding(0, 1)
)
