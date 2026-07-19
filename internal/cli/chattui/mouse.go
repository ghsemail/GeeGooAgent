package chattui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// NormalizeMouseMode returns a Hermes-compatible mouse preset.
func NormalizeMouseMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "off", "false", "0", "no":
		return "off"
	case "wheel":
		return "wheel"
	case "buttons":
		return "buttons"
	case "all", "on", "true", "1", "yes", "":
		return "all"
	case "toggle":
		return "toggle"
	default:
		return "wheel"
	}
}

// CycleMouseMode advances off → wheel → buttons → all → off.
func CycleMouseMode(mode string) string {
	switch NormalizeMouseMode(mode) {
	case "off":
		return "wheel"
	case "wheel":
		return "buttons"
	case "buttons":
		return "all"
	default:
		return "off"
	}
}

// ProgramOptions builds tea.ProgramOption list from display mouse and alt-screen settings.
func ProgramOptions(mouseMode string, altScreen bool) []tea.ProgramOption {
	var opts []tea.ProgramOption
	if altScreen {
		opts = append(opts, tea.WithAltScreen())
	}
	switch NormalizeMouseMode(mouseMode) {
	case "off":
		// no mouse
	case "wheel":
		opts = append(opts, tea.WithMouseCellMotion())
	case "buttons", "all":
		opts = append(opts, tea.WithMouseAllMotion())
	}
	return opts
}
