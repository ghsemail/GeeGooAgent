package strategydev

// multiTimeframeSpecsForKlinePatternDisplay returns exactly four timeframe rows for
// diagnose JSON (Monday 运营 Web 方案 B). Independent of price-analysis hard gates.
func multiTimeframeSpecsForKlinePatternDisplay(signalFreq string) []struct {
	Label string
	Freq  string
} {
	gate := NormalizeFrequency(signalFreq)
	switch gate {
	case "5m":
		return []struct {
			Label string
			Freq  string
		}{
			{"5m", "5m"},
			{"30m", "30m"},
			{"60m", "60m"},
			{"日线", "daily"},
		}
	case "60m":
		return []struct {
			Label string
			Freq  string
		}{
			{"60m", "60m"},
			{"30m", "30m"},
			{"日线", "daily"},
			{"周线", "weekly"},
		}
	case "15m":
		return []struct {
			Label string
			Freq  string
		}{
			{"15m", "15m"},
			{"30m", "30m"},
			{"60m", "60m"},
			{"日线", "daily"},
		}
	default:
		return klinePatternDisplaySpecsFillFour(gate)
	}
}

var klinePatternDisplayLadder = []struct {
	Label string
	Freq  string
}{
	{"5m", "5m"},
	{"15m", "15m"},
	{"30m", "30m"},
	{"60m", "60m"},
	{"日线", "daily"},
	{"周线", "weekly"},
}

func klinePatternDisplaySpecsFillFour(gate string) []struct {
	Label string
	Freq  string
} {
	if gate == "" {
		return multiTimeframeSpecsForKlinePatternDisplay("5m")
	}
	start := -1
	for i, spec := range klinePatternDisplayLadder {
		if spec.Freq == gate {
			start = i
			break
		}
	}
	var out []struct {
		Label string
		Freq  string
	}
	if start < 0 {
		out = append(out, struct {
			Label string
			Freq  string
		}{Label: timeframeDisplayLabel(gate), Freq: gate})
		start = 0
	} else {
		out = append(out, klinePatternDisplayLadder[start])
	}
	for i := start + 1; i < len(klinePatternDisplayLadder) && len(out) < 4; i++ {
		out = append(out, klinePatternDisplayLadder[i])
	}
	for i := start - 1; i >= 0 && len(out) < 4; i-- {
		out = append(out, klinePatternDisplayLadder[i])
	}
	return out
}
