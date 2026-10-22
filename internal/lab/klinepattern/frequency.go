package klinepattern

import "strings"

// NormalizeFrequency maps UI labels to probe frequency codes (copy of strategydev mapping).
func NormalizeFrequency(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch s {
	case "5", "5m", "5min", "5分钟", "5 分钟":
		return "5m"
	case "15m", "15min", "15分钟":
		return "15m"
	case "30m", "30min", "30分钟":
		return "30m"
	case "4h", "4hour":
		return "4h"
	case "240m", "240min":
		return "240m"
	case "60", "60m", "60min", "60分钟", "1h", "1hour":
		return "60m"
	case "daily", "day", "1d", "d", "日线", "日":
		return "daily"
	case "weekly", "1wk", "week", "周线", "周":
		return "weekly"
	default:
		return strings.TrimSpace(raw)
	}
}

// CapMonthsBack limits analysis window to half a year.
func CapMonthsBack(months int) int {
	if months <= 0 {
		return 3
	}
	if months > 6 {
		return 6
	}
	return months
}

func timeframeDisplayLabel(freq string) string {
	switch NormalizeFrequency(freq) {
	case "5m":
		return "5m"
	case "15m":
		return "15m"
	case "30m":
		return "30m"
	case "60m":
		return "60m"
	case "4h":
		return "4h"
	case "daily":
		return "日线"
	case "weekly", "1wk":
		return "周线"
	default:
		return strings.TrimSpace(freq)
	}
}
