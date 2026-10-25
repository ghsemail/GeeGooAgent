package klinepattern

// KlinePatternMTFWeightSchema documents the base MTF weight table (§5.4); shorter ladders renormalize these.
const KlinePatternMTFWeightSchema = "mtf_v1_0.4_0.3_0.2_0.1_renorm"

// KlinePatternMTFWeights are base ladder index weights for a full four-TF row set (signal heaviest).
// Shorter ladders use the first n entries renormalized to sum 1 (see klinePatternMTFWeightsForLadder).
var KlinePatternMTFWeights = []float64{0.4, 0.3, 0.2, 0.1}

type klinePatternMTFSpec struct {
	Label string
	Freq  string
}

// klinePatternMultiTimeframeSpecs matches internal/lab/priceanalysis multiTimeframeSpecs (5m / 1小时 / 日线 / 周线).
var klinePatternMultiTimeframeSpecs = []klinePatternMTFSpec{
	{"5m", "5m"},
	{"1小时", "60m"},
	{"日线", "daily"},
	{"周线", "weekly"},
}

// klinePatternGateFrequencyRank orders timeframes for MTF ladder filtering (higher = coarser).
func klinePatternGateFrequencyRank(freq string) int {
	switch NormalizeFrequency(freq) {
	case "5m":
		return 10
	case "15m":
		return 20
	case "60m":
		return 30
	case "daily":
		return 40
	case "weekly", "1wk":
		return 50
	default:
		return 30
	}
}

// klinePatternMinRankForGate mirrors priceanalysis priceAnalysisMinRank (15m → 60m floor).
func klinePatternMinRankForGate(gateFreq string) int {
	gate := NormalizeFrequency(gateFreq)
	switch gate {
	case "15m":
		return klinePatternGateFrequencyRank("60m")
	default:
		return klinePatternGateFrequencyRank(gate)
	}
}

// multiTimeframeSpecsForKlinePatternDisplay returns signal period plus coarser TFs only,
// aligned with priceanalysis multiTimeframeSpecsForGate (never finer than signal frequency).
func multiTimeframeSpecsForKlinePatternDisplay(signalFreq string) []klinePatternMTFSpec {
	gate := NormalizeFrequency(signalFreq)
	if gate == "" {
		return append([]klinePatternMTFSpec(nil), klinePatternMultiTimeframeSpecs...)
	}
	minRank := klinePatternMinRankForGate(signalFreq)
	var out []klinePatternMTFSpec
	for _, spec := range klinePatternMultiTimeframeSpecs {
		if klinePatternGateFrequencyRank(spec.Freq) >= minRank {
			out = append(out, spec)
		}
	}
	return out
}

func klinePatternMTFWeightsForLadder(n int) []float64 {
	if n <= 0 {
		return nil
	}
	if n > len(KlinePatternMTFWeights) {
		n = len(KlinePatternMTFWeights)
	}
	sub := KlinePatternMTFWeights[:n]
	var sum float64
	for _, w := range sub {
		sum += w
	}
	if sum <= 0 {
		return nil
	}
	out := make([]float64, n)
	for i, w := range sub {
		out[i] = w / sum
	}
	return out
}

// klinePatternProbeFrequency maps MTF ladder freq to probe_bot_signal_series tokens.
func klinePatternProbeFrequency(freq string) string {
	return NormalizeFrequency(freq)
}

// klinePatternMTFTimeframeKey canonicalizes ladder slot ids.
func klinePatternMTFTimeframeKey(freq string) string {
	return NormalizeFrequency(freq)
}
