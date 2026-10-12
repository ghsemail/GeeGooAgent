package strategydev

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// NormalizeFrequency maps UI labels to probe frequency codes.
func NormalizeFrequency(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch s {
	case "5", "5m", "5min", "5分钟", "5 分钟":
		return "5m"
	case "15m", "15min", "15分钟":
		return "15m"
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

// BarLimitForFrequency returns fixed bar counts for step 1 (includes warmup).
func BarLimitForFrequency(freq string) int {
	switch NormalizeFrequency(freq) {
	case "5m":
		return 200
	case "60m":
		return 200
	case "daily":
		return 120
	case "weekly", "1wk":
		return 30
	default:
		return 200
	}
}

// PriceAnalysisProbeArgs builds probe_bot_signal_series args for multi-TF price analysis.
// Omits limit so GeeGooSignal derives bar span from months_back (not “latest N from now”).
func PriceAnalysisProbeArgs(code, freq string, months int) map[string]any {
	// Normalize so probe_bot_signal_series schema validation accepts nosignal placeholders.
	buy := slots.NormalizeSignalChain(nosignalProbe)
	sell := slots.NormalizeSignalChain(nosignalProbe)
	return map[string]any{
		"code":        strings.TrimSpace(code),
		"frequency":   NormalizeFrequency(freq),
		"months_back": CapMonthsBack(months),
		"buy_signal":  buy,
		"sell_signal": sell,
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
