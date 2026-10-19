package strategydev

// KlinePatternMTFWeightSchema documents the locked MTF weight table (§5.4).
const KlinePatternMTFWeightSchema = "mtf_v1_0.4_0.3_0.2_0.1"

// KlinePatternMTFWeights are ladder index weights (signal TF heaviest); sum = 1.
var KlinePatternMTFWeights = []float64{0.4, 0.3, 0.2, 0.1}

type klinePatternMTFSpec struct {
	Label string
	Freq  string
}

// multiTimeframeSpecsForKlinePatternDisplay returns exactly four timeframe rows for
// diagnose K-line MTF JSON. Independent of price-analysis hard gates (§4).
func multiTimeframeSpecsForKlinePatternDisplay(signalFreq string) []klinePatternMTFSpec {
	gate := NormalizeFrequency(signalFreq)
	switch gate {
	case "5m":
		return []klinePatternMTFSpec{
			{"5m", "5m"},
			{"15m", "15m"},
			{"30m", "30m"},
			{"60m", "60m"},
		}
	case "60m":
		return []klinePatternMTFSpec{
			{"1h", "60m"},
			{"4h", "4h"},
			{"日线", "daily"},
			{"周线", "weekly"},
		}
	case "15m":
		return []klinePatternMTFSpec{
			{"15m", "15m"},
			{"30m", "30m"},
			{"60m", "60m"},
			{"日线", "daily"},
		}
	default:
		return klinePatternDisplaySpecsFillFour(gate)
	}
}

var klinePatternDisplayLadder = []klinePatternMTFSpec{
	{"5m", "5m"},
	{"15m", "15m"},
	{"30m", "30m"},
	{"60m", "60m"},
	{"4h", "4h"},
	{"日线", "daily"},
	{"周线", "weekly"},
}

func klinePatternDisplaySpecsFillFour(gate string) []klinePatternMTFSpec {
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
	var out []klinePatternMTFSpec
	if start < 0 {
		out = append(out, klinePatternMTFSpec{Label: timeframeDisplayLabel(gate), Freq: gate})
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

// klinePatternProbeFrequency maps MTF ladder freq to probe_bot_signal_series tokens.
func klinePatternProbeFrequency(freq string) string {
	switch NormalizeFrequency(freq) {
	case "4h":
		return "240m"
	case "240m":
		return "240m"
	default:
		return NormalizeFrequency(freq)
	}
}

// klinePatternMTFTimeframeKey canonicalizes ladder slot ids (4h and 240m share a slot).
func klinePatternMTFTimeframeKey(freq string) string {
	switch NormalizeFrequency(freq) {
	case "240m":
		return "4h"
	default:
		return NormalizeFrequency(freq)
	}
}
