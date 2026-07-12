package chattui

import "strings"

// EstimateBlockHeight returns approximate terminal rows for a block.
func EstimateBlockHeight(b Block, expanded bool) int {
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
		if !expanded {
			return 1
		}
		body := strings.TrimRight(b.Body, "\n")
		if body == "" {
			return 1
		}
		return 1 + strings.Count(body, "\n") + 1
	}
}

// EstimateTranscriptHeight sums visible block heights.
func EstimateTranscriptHeight(blocks []Block, displayMode func(Block) (visible, expanded bool)) int {
	total := 0
	for _, b := range blocks {
		vis, exp := displayMode(b)
		if !vis {
			continue
		}
		total += EstimateBlockHeight(b, exp)
	}
	return total
}
