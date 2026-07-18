package chattui

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// EstimateBlockHeight returns approximate terminal rows for a block.
func EstimateBlockHeight(b Block, cfg config.DisplayConfig) int {
	switch b.Kind {
	case KindUser:
		return 1 + strings.Count(b.Body, "\n")
	case KindReply:
		body := strings.TrimRight(b.Body, "\n")
		if body == "" {
			return 1
		}
		return 1 + strings.Count(body, "\n") + 1
	default:
		if b.IsExpanded(cfg) {
			body := strings.TrimRight(b.Body, "\n")
			if body == "" {
				return 1
			}
			return 1 + strings.Count(body, "\n") + 1
		}
		if b.ShowThinkingPreview(cfg) {
			n := len(b.LastBodyLines(thinkingPreviewLines))
			if n == 0 {
				return 1
			}
			return 1 + n
		}
		if b.ShowLivePreview(cfg) {
			return 2
		}
		return 1
	}
}

// EstimateTranscriptHeight sums visible block heights.
func EstimateTranscriptHeight(blocks []Block, cfg config.DisplayConfig) int {
	total := 0
	for i := 0; i < len(blocks); {
		b := blocks[i]
		if !b.IsVisible(cfg) {
			i++
			continue
		}
		if IsProcessKind(b.Kind) {
			group := 2 // process panel border
			for i < len(blocks) && blocks[i].IsVisible(cfg) && IsProcessKind(blocks[i].Kind) {
				group += EstimateBlockHeight(blocks[i], cfg)
				i++
			}
			total += group
			continue
		}
		total += EstimateBlockHeight(b, cfg)
		i++
	}
	return total
}
