package chatui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var heroLines = []string{
	"  ╔═╗ ╔═╗ ╔═╗   ╔═╗ ╔═╗ ╔═╗",
	"  ║ ╚╗ ║  ║     ║ ╔╝ ║ ║ ║ ║",
	"  ║ ╔╝ ╚═╗ ╚═╗   ║ ╚╗ ║ ║ ║ ║",
	"  ╚═╝  ╚═╝ ╚═╝   ╚═╝ ╚═╝ ╚═╝",
}

// wideLogoLines spells GEEGOO in block letters (display width ~50 cols).
var wideLogoLines = []string{
	" ██████╗ ███████╗███████╗ ██████╗  █████╗  █████╗",
	"██╔════╝ ██╔════╝██╔════╝██╔════╝ ██╔══██╗██╔══██╗",
	"██║  ███╗█████╗  █████╗  ██║  ███╗██║  ██║██║  ██║",
	"██║   ██║██╔══╝  ██╔══╝  ██║   ██║██║  ██║██║  ██║",
	" ╚██████╔╝ ███████╗███████╗ ╚██████╔╝ ╚█████╔╝ ╚█████╔╝",
	" ╚═════╝ ╚══════╝╚══════╝ ╚═════╝  ╚════╝  ╚════╝",
}

func shouldShowWideLogo(terminalWidth int) bool {
	if terminalWidth <= 0 {
		return true
	}
	return terminalWidth >= wideLogoDisplayWidth()+2
}

func wideLogoDisplayWidth() int {
	max := 0
	for i, line := range wideLogoLines {
		styled := styleTitle.Render(line)
		if i%2 != 0 {
			styled = styleBrand.Render(line)
		}
		if w := lipgloss.Width(styled); w > max {
			max = w
		}
	}
	return max
}

// renderBannerLogo returns the top header block (block GEEGOO or compact hero), left-aligned.
func renderBannerLogo(terminalWidth int) string {
	if shouldShowWideLogo(terminalWidth) {
		return renderWideLogo()
	}
	var b strings.Builder
	b.WriteString(styleBrand.Render(agentTitle + " Agent"))
	b.WriteByte('\n')
	b.WriteString(renderHero())
	return b.String()
}

func renderWideLogo() string {
	var out strings.Builder
	for i, line := range wideLogoLines {
		if i > 0 {
			out.WriteByte('\n')
		}
		if i%2 == 0 {
			out.WriteString(styleTitle.Render(line))
		} else {
			out.WriteString(styleBrand.Render(line))
		}
	}
	return out.String()
}

func renderHero() string {
	var out strings.Builder
	for i, line := range heroLines {
		if i > 0 {
			out.WriteByte('\n')
		}
		if i%2 == 0 {
			out.WriteString(styleTitle.Render(line))
		} else {
			out.WriteString(styleBrand.Render(line))
		}
	}
	return out.String()
}
