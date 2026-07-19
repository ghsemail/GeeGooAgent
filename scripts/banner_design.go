//go:build ignore

// Plain-text banner mockup for review. Run: go run scripts/banner_design.go [width]
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	w := 80
	if len(os.Args) > 1 {
		fmt.Sscanf(os.Args[1], "%d", &w)
	}
	fmt.Println(designAtWidth(w))
}

func designAtWidth(w int) string {
	logo := []string{
		" ██████╗ ███████╗███████╗ ██████╗  █████╗  █████╗",
		"██╔════╝ ██╔════╝██╔════╝██╔════╝ ██╔══██╗██╔══██╗",
		"██║  ███╗█████╗  █████╗  ██║  ███╗██║  ██║██║  ██║",
		"██║   ██║██╔══╝  ██╔══╝  ██║   ██║██║  ██║██║  ██║",
		" ╚██████╔╝ ███████╗███████╗ ╚██████╔╝ ╚█████╔╝ ╚█████╔╝",
		" ╚═════╝ ╚══════╝╚══════╝ ╚═════╝  ╚════╝  ╚════╝",
	}
	var b strings.Builder
	b.WriteString(strings.Repeat("─", w))
	b.WriteByte('\n')
	b.WriteString("GeeGoo Agent 启动 Banner 设计稿（左对齐，宽屏 ≥56 列显示大字）\n")
	b.WriteString(strings.Repeat("─", w))
	b.WriteByte('\n')
	for _, line := range logo {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteString("GeeGoo Agent · upstream 3c158363\n")
	b.WriteString("╭" + strings.Repeat("─", w-2) + "╮\n")
	left := []string{
		"deepseek-chat · deepseek",
		"think on · dry-run off",
		"/home/ubuntu/.geegoo/geegoo-agent",
		"Session: sess-…",
	}
	right := []string{
		"Available Tools",
		"perceive: search_code",
		"analyze: get_current_price, get_mcp_analysis, …",
		"decide: generate_dca_strategy, …",
		"act: create_dca_bot, create_grid_bot",
		"11 tools · 0 skills · /help",
	}
	for i := 0; i < len(right); i++ {
		l := ""
		if i < len(left) {
			l = left[i]
		}
		b.WriteString(fmt.Sprintf("│ %-28s %-*s │\n", l, w-33, right[i]))
	}
	b.WriteString("╰" + strings.Repeat("─", w-2) + "╯\n")
	b.WriteString("Welcome to GeeGoo Agent!  Type your message or /help for commands.\n")
	b.WriteString("✦ Tips:\n")
	b.WriteString("  · /details collapsed 折叠思考与工具 · /help 查看命令\n")
	b.WriteString(strings.Repeat("─", w))
	b.WriteByte('\n')
	b.WriteString("窄屏 (<56 列) 回退：\n")
	b.WriteString("⚕ GeeGoo Agent\n")
	b.WriteString("  ╔═╗ ╔═╗ ╔═╗   ╔═╗ ╔═╗ ╔═╗\n")
	b.WriteString("  ║ ╚╗ ║  ║     ║ ╔╝ ║ ║ ║ ║\n")
	b.WriteString("  ║ ╔╝ ╚═╗ ╚═╗   ║ ╚╗ ║ ║ ║ ║\n")
	b.WriteString("  ╚═╝  ╚═╝ ╚═╝   ╚═╝ ╚═╝ ╚═╝\n")
	return b.String()
}
