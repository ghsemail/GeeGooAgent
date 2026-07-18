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
		if b.ShowLivePreview(cfg) {
			return 2
		}
		return 1
	}
}

// EstimateTranscriptHeight sums visible block heights.
func EstimateTranscriptHeight(blocks []Block, cfg config.DisplayConfig) int {
	total := 0
	for _, b := range blocks {
		if !b.IsVisible(cfg) {
			continue
		}
		total += EstimateBlockHeight(b, cfg)
	}
	return total
}
