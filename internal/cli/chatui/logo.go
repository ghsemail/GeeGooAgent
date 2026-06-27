package chatui

import "strings"

var heroLines = []string{
	"  ╔═╗ ╔═╗ ╔═╗   ╔═╗ ╔═╗ ╔═╗",
	"  ║ ╚╗ ║  ║     ║ ╔╝ ║ ║ ║ ║",
	"  ║ ╔╝ ╚═╗ ╚═╗   ║ ╚╗ ║ ║ ║ ║",
	"  ╚═╝  ╚═╝ ╚═╝   ╚═╝ ╚═╝ ╚═╝",
}

var wideLogoLines = []string{
	" ██████╗ ███████╗███████╗ ██████╗  █████╗  █████╗",
	"██╔════╝ ██╔════╝██╔════╝██╔════╝ ██╔══██╗██╔══██╗",
	"██║  ███╗█████╗  █████╗  ██║  ███╗██║  ██║██║  ██║",
	"██║   ██║██╔══╝  ██╔══╝  ██║   ██║██║  ██║██║  ██║",
	"╚██████╔╝███████╗███████╗╚██████╔╝╚█████╔╝╚█████╔╝",
	" ╚═════╝ ╚══════╝╚══════╝ ╚═════╝  ╚════╝  ╚════╝",
}

func renderHero() string {
	var out strings.Builder
	for i, line := range heroLines {
		if i%2 == 0 {
			out.WriteString(styleAccent.Render(line))
		} else {
			out.WriteString(styleGold.Render(line))
		}
		out.WriteByte('\n')
	}
	return strings.TrimRight(out.String(), "\n")
}

func renderWideLogo() string {
	var out strings.Builder
	for i, line := range wideLogoLines {
		if i%2 == 0 {
			out.WriteString(styleAccent.Render(line))
		} else {
			out.WriteString(styleGold.Render(line))
		}
		out.WriteByte('\n')
	}
	return strings.TrimRight(out.String(), "\n")
}
