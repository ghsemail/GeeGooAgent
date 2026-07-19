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

func renderHero() string {
	var out strings.Builder
	for i, line := range heroLines {
		if i%2 == 0 {
			out.WriteString(styleTitle.Render(line))
		} else {
			out.WriteString(styleBrand.Render(line))
		}
		out.WriteByte('\n')
	}
	return strings.TrimRight(out.String(), "\n")
}

func renderWideLogo() string {
	return renderWideLogoForWidth(0)
}

// renderWideLogoForWidth draws the block GEEGOO logo, centered when terminalWidth allows.
// Falls back to the compact hero when the terminal is too narrow.
func renderWideLogoForWidth(terminalWidth int) string {
	logoW := wideLogoDisplayWidth()
	if !shouldShowWideLogo(terminalWidth) {
		return renderHero()
	}
	leftPad := 0
	if terminalWidth > logoW {
		leftPad = (terminalWidth - logoW) / 2
	}
	pad := strings.Repeat(" ", leftPad)
	var out strings.Builder
	for i, line := range wideLogoLines {
		if i > 0 {
			out.WriteByte('\n')
		}
		styled := styleTitle.Render(line)
		if i%2 != 0 {
			styled = styleBrand.Render(line)
		}
		out.WriteString(pad)
		out.WriteString(styled)
	}
	return out.String()
}
