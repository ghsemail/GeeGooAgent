package chatcmd

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// DetailsResult is the outcome of parsing /details.
type DetailsResult struct {
	OK          bool
	Persist     bool
	ShowLast    bool
	Message     string
	Display     *config.DisplayConfig
}

// ApplyDetails mutates display according to `/details ...` args (without the command word).
// args is the remainder after "/details", already split on spaces.
func ApplyDetails(display *config.DisplayConfig, args []string) DetailsResult {
	if display == nil {
		return DetailsResult{OK: false, Message: "display config is nil"}
	}
	d := *display
	d.Normalize()

	if len(args) == 0 {
		return DetailsResult{
			OK: true, Persist: false, Display: &d,
			Message: formatDetailsStatus(d),
		}
	}

	a0 := strings.ToLower(strings.TrimSpace(args[0]))
	switch a0 {
	case "last":
		return DetailsResult{OK: true, ShowLast: true, Display: &d, Message: "expanding last thinking/tools"}
	case "cycle":
		d.DetailsMode = config.CycleDetailsMode(d.DetailsMode)
		return DetailsResult{OK: true, Persist: true, Display: &d, Message: "details_mode=" + d.DetailsMode}
	case config.ModeHidden, config.ModeCollapsed, config.ModeExpanded:
		d.DetailsMode = a0
		return DetailsResult{OK: true, Persist: true, Display: &d, Message: "details_mode=" + d.DetailsMode}
	case "thinking", "tools", "activity":
		if len(args) < 2 {
			return DetailsResult{OK: false, Display: &d, Message: "usage: /details " + a0 + " [hidden|collapsed|expanded|reset]"}
		}
		mode := strings.ToLower(strings.TrimSpace(args[1]))
		if mode == "reset" {
			mode = ""
		} else if mode != config.ModeHidden && mode != config.ModeCollapsed && mode != config.ModeExpanded {
			return DetailsResult{OK: false, Display: &d, Message: "invalid mode: " + mode}
		}
		switch a0 {
		case "thinking":
			d.Sections.Thinking = mode
		case "tools":
			d.Sections.Tools = mode
		case "activity":
			d.Sections.Activity = mode
			if mode == "" {
				d.Sections.Activity = config.ModeHidden
			}
		}
		d.Normalize()
		return DetailsResult{OK: true, Persist: true, Display: &d, Message: a0 + "=" + d.EffectiveMode(a0)}
	default:
		return DetailsResult{OK: false, Display: &d, Message: "usage: /details [hidden|collapsed|expanded|cycle|last] or /details <section> <mode>"}
	}
}

func formatDetailsStatus(d config.DisplayConfig) string {
	return "details_mode=" + d.DetailsMode +
		" thinking=" + d.EffectiveMode("thinking") +
		" tools=" + d.EffectiveMode("tools") +
		" activity=" + d.EffectiveMode("activity")
}
