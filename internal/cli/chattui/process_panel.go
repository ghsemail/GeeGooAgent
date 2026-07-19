package chattui

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatui"
)

func (m *Model) appendProcessBlock(b *strings.Builder, block Block, blockIdx, focus int, width int) {
	prefix := ""
	if blockIdx == focus {
		prefix = styleFocus.Render("› ")
	}
	b.WriteString(prefix + renderSectionHeader(block, m.display))
	b.WriteByte('\n')
	if block.IsExpanded(m.display) {
		body := strings.TrimRight(block.Body, "\n")
		for _, line := range strings.Split(body, "\n") {
			if strings.TrimSpace(line) == "" {
				b.WriteByte('\n')
				continue
			}
			if block.Kind == KindThinking {
				b.WriteString(chatui.RenderThinkingLineWidth(line, width))
			} else {
				b.WriteString(chatui.RenderDetailLineWidth(line, width))
			}
			b.WriteByte('\n')
		}
		return
	}
	if block.ShowThinkingPreview(m.display) {
		for _, line := range block.LastBodyLines(collapsedPreviewLines) {
			b.WriteString(chatui.RenderThinkingLineWidth(line, width))
			b.WriteByte('\n')
		}
		return
	}
	if block.ShowToolPreview(m.display) {
		for _, line := range block.LastBodyLines(collapsedPreviewLines) {
			b.WriteString(chatui.RenderDetailLineWidth(line, width))
			b.WriteByte('\n')
		}
		return
	}
}
