package chatui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// ConfigureTextInput applies the GeeGoo white/yellow palette to a bubbles input.
// Bubbles defaults and unset cursor text styles can render as violet on some terminals.
func ConfigureTextInput(ti *textinput.Model) {
	if ti == nil {
		return
	}
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorText))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorDim))
	goldStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGold))

	ti.TextStyle = textStyle
	ti.PlaceholderStyle = dimStyle
	ti.CompletionStyle = goldStyle
	ti.PromptStyle = goldStyle
	ti.Cursor.Style = goldStyle
	ti.Cursor.TextStyle = textStyle
	ti.ShowSuggestions = true
	ti.SetSuggestions(SlashCommandStrings())
}
