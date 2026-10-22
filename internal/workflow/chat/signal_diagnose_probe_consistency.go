package chat

// diagnosePayloadConsistentWithProbe rejects diagnose shells whose merged series are all zero
// while probe already recorded non-zero Signal-API hits (00700-style mismatch).
func diagnosePayloadConsistentWithProbe(flow *Flow, data map[string]any) bool {
	if flow == nil || data == nil {
		return true
	}
	if flow.ProbeBuyHitsSignalAPI <= 0 && flow.ProbeSellHitsSignalAPI <= 0 {
		return true
	}
	diagBuy := probeHitCount(data, "buy_hits", "buy_merged", 1)
	diagSell := probeHitCount(data, "sell_hits", "sell_merged", -1)
	if hits, ok := data["hits"].(map[string]any); ok {
		if diagBuy == 0 {
			diagBuy = intAny(hits["buy"])
		}
		if diagSell == 0 {
			diagSell = intAny(hits["sell"])
		}
	}
	if flow.ProbeBuyHitsSignalAPI > 0 && diagBuy == 0 && flow.ProbeSellHitsSignalAPI > 0 && diagSell == 0 {
		return false
	}
	return true
}
