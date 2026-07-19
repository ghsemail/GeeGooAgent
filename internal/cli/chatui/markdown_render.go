package chatui

import (
	"strings"
)

// AssistantRenderOptions controls assistant reply rendering.
type AssistantRenderOptions struct {
	Markdown bool
	Live     bool
}

// RenderAssistantBox renders a completed assistant reply.
func RenderAssistantBox(text string, width int) string {
	return RenderAssistantBoxWith(text, width, AssistantRenderOptions{Markdown: true})
}

// RenderAssistantBoxLive renders a streaming assistant preview (plain text).
func RenderAssistantBoxLive(text string, width int) string {
	return RenderAssistantBoxWith(text, width, AssistantRenderOptions{Live: true})
}

// RenderAssistantBoxWith renders assistant text with the given options.
// Live means streaming typewriter preview only; completed replies use Markdown when enabled.
func RenderAssistantBoxWith(text string, width int, opts AssistantRenderOptions) string {
	if width <= 0 {
		width = 80
	}
	body := strings.TrimRight(text, "\n")
	if strings.TrimSpace(body) == "" {
		if opts.Live {
			return RenderWorkingLine()
		}
		return styleMeta.Render("⋯ 正在生成回复…")
	}
	if opts.Live {
		return RenderGrokReplyBlock(body, width)
	}
	panelW := assistantBoxOuterWidth(width)
	innerW := PanelContentWidth(width)
	var inner string
	if opts.Markdown {
		inner = RenderAssistantMarkdownAt(body, innerW)
	} else {
		inner = RenderPlainAssistantBody(body, innerW)
	}
	if strings.TrimSpace(stripANSI(inner)) == "" {
		return styleMeta.Render("⋯ 正在生成回复…")
	}
	return stylePanel.Width(panelW).Render(inner)
}

// RenderAssistantMarkdown renders markdown via glamour with terminal word-wrap.
func RenderAssistantMarkdown(text string, width int) string {
	return RenderAssistantMarkdownAt(text, ContentWrapWidth(width))
}

// RenderAssistantMarkdownAt renders markdown at a fixed inner column width.
func RenderAssistantMarkdownAt(text string, innerW int) string {
	text = strings.TrimRight(text, "\n")
	if strings.TrimSpace(text) == "" {
		return styleMeta.Render("⋯ 正在生成回复…")
	}
	text = PreprocessTerminalMarkdown(text)
	w := innerW
	if w < 32 {
		w = 32
	}
	r, err := newMarkdownRenderer(w)
	if err != nil {
		return RenderPlainAssistantBody(text, w)
	}
	out, err := r.Render(text)
	if err != nil || markdownLooksUnrendered(out, text) {
		return RenderPlainAssistantBody(text, w)
	}
	return strings.TrimRight(out, "\n")
}

func markdownLooksUnrendered(rendered, source string) bool {
	plain := strings.ReplaceAll(rendered, "\x1b[0m", "")
	if strings.Contains(plain, "##") && strings.Contains(source, "##") {
		return true
	}
	if strings.Contains(plain, "**") && strings.Contains(source, "**") {
		return true
	}
	return false
}
