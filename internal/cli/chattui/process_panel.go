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
			if block.Kind == KindThinking {
				b.WriteString(chatui.RenderThinkingLine(line))
			} else {
				b.WriteString(chatui.RenderDetailLine(line))
			}
			b.WriteByte('\n')
		}
		return
	}
	if block.ShowThinkingPreview(m.display) {
		for _, line := range block.LastBodyLines(thinkingPreviewLines) {
			b.WriteString(chatui.RenderThinkingLine(TruncateRunes(line, width-8)))
			b.WriteByte('\n')
		}
		return
	}
	if block.ShowLivePreview(m.display) {
		line := TruncateRunes(block.LastBodyLine(), width-8)
		b.WriteString(chatui.RenderDetailLine(line))
		b.WriteByte('\n')
	}
}

func (m *Model) renderProcessGroup(blocks []Block, start, focus int, width int) string {
	var panel strings.Builder
	for i := start; i < len(blocks); i++ {
		block := blocks[i]
		if !block.IsVisible(m.display) || !IsProcessKind(block.Kind) {
			break
		}
		if panel.Len() > 0 {
			panel.WriteByte('\n')
		}
		m.appendProcessBlock(&panel, block, i, focus, width)
	}
	return chatui.RenderProcessPanel(panel.String(), width)
}
