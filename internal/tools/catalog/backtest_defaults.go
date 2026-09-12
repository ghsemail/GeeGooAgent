package catalog

import (
	"fmt"
	"strings"
)

// ApplyStrategyBacktestDefaults fills months_back/period when the model omits them.
// Matches GeeGooSignal /runStrategyBacktest (default 3 months).
func ApplyStrategyBacktestDefaults(body map[string]any) {
	if body == nil {
		return
	}
	months, hasMonths := intish(body["months_back"])
	period, _ := body["period"].(string)
	period = strings.TrimSpace(period)
	if (!hasMonths || months <= 0) && period == "" {
		body["months_back"] = 3
		body["period"] = "3m"
		return
	}
	if period == "" && hasMonths && months > 0 {
		body["period"] = fmt.Sprintf("%dm", months)
	}
}

func intish(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	case float32:
		return int(n), true
	default:
		return 0, false
	}
}
