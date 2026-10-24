package klinepattern

// mtfBundleSeries returns probe data for one ladder freq (canonical ByFreq keys).
func mtfBundleSeries(bundle *KlinePatternMTFBundle, ladderFreq string) (KlinePatternFreqSeries, bool) {
	if bundle == nil || bundle.ByFreq == nil {
		return KlinePatternFreqSeries{}, false
	}
	key := klinePatternMTFTimeframeKey(ladderFreq)
	if ser, ok := bundle.ByFreq[ladderFreq]; ok {
		return ser, true
	}
	if key != ladderFreq {
		if ser, ok := bundle.ByFreq[key]; ok {
			return ser, true
		}
	}
	return KlinePatternFreqSeries{}, false
}
