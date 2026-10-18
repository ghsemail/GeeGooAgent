package priceanalysis

import "time"

func truncateBarsAtAnchor(bars []any, anchorTime time.Time) []any {
	if len(bars) == 0 {
		return nil
	}
	if anchorTime.IsZero() {
		return bars
	}
	out := make([]any, 0, len(bars))
	for _, raw := range bars {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ts := barTimeUnix(row)
		if ts.IsZero() || !ts.After(anchorTime) {
			out = append(out, raw)
		}
	}
	return out
}
