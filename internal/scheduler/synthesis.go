package scheduler

import "strings"

func skillRequiresSynthesis(skill string) bool {
	switch strings.TrimSpace(skill) {
	case "premarket_market", "premarket_stock", "postmarket_stock":
		return true
	default:
		return false
	}
}
