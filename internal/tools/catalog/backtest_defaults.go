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
	months, hasMonths := IntFromAny(body["months_back"])
	if hasMonths && months > 0 {
		body["months_back"] = months
	}
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

