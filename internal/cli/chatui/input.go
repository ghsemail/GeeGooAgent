package chatui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// ConfigureTextInput applies the GeeGoo palette to a bubbles input.
func ConfigureTextInput(ti *textinput.Model) {
	if ti == nil {
		return
	}
	ti.TextStyle = styleBody
	ti.PlaceholderStyle = styleWhisper
	ti.CompletionStyle = styleTitle
	ti.PromptStyle = lipgloss.NewStyle()
	ti.Cursor.Style = styleBrand
	ti.Cursor.TextStyle = styleBody
	ti.ShowSuggestions = true
	ti.SetSuggestions(SlashCommandStrings())
}
