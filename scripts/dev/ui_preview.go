//go:build ignore

package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatui"
)

func section(title string) {
	fmt.Println()
	fmt.Println(strings.Repeat("═", 72))
	fmt.Printf("  %s\n", title)
	fmt.Println(strings.Repeat("═", 72))
}

func main() {
	w := 100
	if len(os.Args) > 1 {
		fmt.Sscanf(os.Args[1], "%d", &w)
	}

	opts := chatui.BannerOptions{
		Provider:  "deepseek",
		Model:     "deepseek-chat",
		SessionID: "sess-demo-001",
		Thinking:  true,
		DryRun:    false,
		Revision:  "demo",
		Workspace: "/home/ubuntu/apps/GeeGooAgent",
		ToolNames: []string{
			"search_code", "get_current_price", "get_mcp_analysis", "get_single_prompt_template",
			"generate_dca_strategy", "generate_grid_strategy", "loopback_strategy",
			"create_dca_bot", "create_grid_bot", "list_dca_bots", "clarify",
		},
		APIHosts: map[string]string{
			"geegoo-bot": "api.example.com",
			"signal":     "signal.example.com",
			"data":       "data.example.com",
		},
	}

	section("1. 启动 Banner（Hermes 完整版：大 Logo + 金边双栏 Tools/Skills + Tips）")
	fmt.Print(chatui.RenderBanner(opts, w, false))

	section("2. 对话区（用户 Grok 条 + 思考/工具软分割 + 正式答复金线 + Agent 标题）")
	width := w
	var conv strings.Builder
	conv.WriteString(chatui.RenderRule(width))
	conv.WriteByte('\n')
	conv.WriteString(chatui.RenderUserPromptBox("分析一下腾讯控股的走势", width))
	conv.WriteByte('\n')
	conv.WriteString(chatui.RenderSoftDivider(width))
	conv.WriteByte('\n')
	conv.WriteString(chatui.RenderGrokProcessHeader(false, "💭 思考", 2, 1.2))
	conv.WriteByte('\n')
	conv.WriteString(chatui.RenderGrokThinkingLine("先 search_code 确认 00700.HK", width))
	conv.WriteByte('\n')
	conv.WriteString(chatui.RenderSoftDivider(width))
	conv.WriteByte('\n')
	conv.WriteString(chatui.RenderGrokProcessHeader(false, "🔧 工具", 2, 0))
	conv.WriteByte('\n')
	conv.WriteString(chatui.RenderGrokToolLine("→ search_code", width))
	conv.WriteByte('\n')
	conv.WriteString(chatui.RenderGrokToolLine("✓ get_mcp_analysis [ok]", width))
	conv.WriteByte('\n')
	conv.WriteString(chatui.RenderAgentHeader(width))
	conv.WriteByte('\n')
	md := "## 腾讯控股 (00700.HK)\n\n**结论**：短期震荡偏多。\n\n- 日线 MACD 金叉\n- 量能温和放大"
	conv.WriteString(chatui.RenderAssistantMarkdown(md, width))
	conv.WriteByte('\n')
	conv.WriteString(chatui.RenderTurnFooter(8200 * time.Millisecond))
	conv.WriteByte('\n')
	fmt.Print(conv.String())

	section("3. Clarify 面板（Hermes 金边，保留）")
	fmt.Println(chatui.RenderClarifyPanel(
		"你要用哪种信号生成 DCA 方案？",
		[]string{"单指标信号", "组合信号"},
		0, w,
	))

	section("4. 底栏（Hermes：模型 · Token 条 · 耗时 · 状态）+ Grok 圆角输入框")
	fmt.Println(chatui.RenderHermesStatusBar(chatui.StatusBarOptions{
		Model: "deepseek/deepseek-chat", PromptTokens: 19600, ContextWindow: 128_000,
		Elapsed: 12 * time.Second, Busy: false, Steps: 3,
	}, w))
	ti := textinput.New()
	ti.Width = 60
	ti.Placeholder = "Type your message or /help for commands."
	chatui.ConfigureTextInput(&ti)
	fmt.Println(chatui.RenderInputChrome(ti.View(), "deepseek/deepseek-chat", w))

	section("5. 排版层级（终端用颜色+字重模拟大小）")
	fmt.Println(`  品牌金 220 bold  — Logo、⚕、金线、聚焦项、用户 >
  标题琥珀 214 bold — Banner 分区、Clarify ?、Markdown 标题
  正文白 252        — 助手/工具正文、选项文字
  用户灰 250        — 用户消息内容
  元信息 244 italic — 思考、统计、chevron
  浅灰 240          — 软分割、占位、页脚、边框`)

	section("6. 布局总览")
	fmt.Print(`
┌─ Viewport ─────────────────────────────────────────────────────┐
│ [Banner] Hermes 大 Logo + 金边双栏（有用户消息后隐藏）            │
│ ──────────────────────────────────────── 金线（用户回合）        │
│ > 用户消息（Grok 灰底条）                                         │
│   ───── 软分割                                                  │
│ ▸ 💭 思考                                                       │
│   ───── 软分割                                                  │
│ ▸ 🔧 工具                                                       │
│ ⚕ GeeGoo ─────────────────── Agent 标题线（无金线）              │
│ ## 助手 Markdown 正式答复                                         │
│ Worked for 8.2s.                                                │
└─────────────────────────────────────────────────────────────────┘
 ⚕ deepseek-chat │ 19.6K/128K [██░░] 15% 12s ✓ 12s 3 steps   ← Hermes 底栏
╭────────────────────────────────────────────────────────────────╮
│ > 输入…                                          deepseek-chat │ ← Grok 输入框
╰────────────────────────────────────────────────────────────────╯
`)
}
