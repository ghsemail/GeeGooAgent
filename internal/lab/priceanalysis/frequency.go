package priceanalysis

import "strings"

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

// LabKlineFetchFrequency maps ladder slots to GeeGooSignal lab-api / probe tokens.
func LabKlineFetchFrequency(freq string) string {
	return NormalizeFrequency(freq)
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

// capMonthsBackForPriceAnalysisFetch applies CapMonthsBack for fine TFs; coarse TFs (weekly)
// may need a longer calendar span so anchor truncate + analysis tail still has enough bars.
func capMonthsBackForPriceAnalysisFetch(freq string, months int) int {
	if months <= 0 {
		months = 3
	}
	switch NormalizeFrequency(freq) {
	case "weekly", "1wk":
		const weeklyFetchMonthsCap = 12
		if months > weeklyFetchMonthsCap {
			return weeklyFetchMonthsCap
		}
		return months
	default:
		return CapMonthsBack(months)
	}
}

// perSignalPriceFetchMonthsCap is the max calendar span for multi-TF lookback probes (取数窗),
// separate from the diagnose analysis window (分析窗) capped at CapMonthsBack.
const perSignalPriceFetchMonthsCap = 12

// capMonthsBackForPerSignalPriceFetch caps extended per-signal lookback fetches (not the 6mo analysis cap).
func capMonthsBackForPerSignalPriceFetch(freq string, months int) int {
	if months <= 0 {
		months = 3
	}
	_ = freq
	if months > perSignalPriceFetchMonthsCap {
		return perSignalPriceFetchMonthsCap
	}
	return months
}
