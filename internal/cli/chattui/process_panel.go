package chattui

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatui"
)

func (m *Model) appendProcessBlock(b *strings.Builder, block Block, blockIdx, focus int, width int) {
	expanded := block.IsExpanded(m.display)
	header := chatui.RenderGrokProcessHeader(expanded, block.Title, block.LineCount(), block.DurationSec)
	if blockIdx == focus {
		header = styleFocus.Render("› ") + header
	}
	b.WriteString(header)
	b.WriteByte('\n')
	if expanded {
		body := strings.TrimRight(block.Body, "\n")
		for _, line := range strings.Split(body, "\n") {
			if strings.TrimSpace(line) == "" {
				b.WriteByte('\n')
				continue
			}
			if block.Kind == KindThinking {
				b.WriteString(chatui.RenderGrokThinkingLine(line, width))
			} else {
				b.WriteString(chatui.RenderGrokToolLine(line, width))
			}
			b.WriteByte('\n')
		}
		return
	}
	if block.ShowThinkingPreview(m.display) {
		for _, line := range block.LastBodyLines(collapsedPreviewLines) {
			b.WriteString(chatui.RenderGrokThinkingLine(line, width))
			b.WriteByte('\n')
		}
		return
	}
	if block.ShowToolPreview(m.display) {
		for _, line := range block.LastBodyLines(collapsedPreviewLines) {
			b.WriteString(chatui.RenderGrokToolLine(line, width))
			b.WriteByte('\n')
		}
	}
}
