package slots

import "strings"

// ShouldResolveNewSymbolForProbe reports whether a signal_probe turn must
// resolve a new stock (search_code) instead of reusing the prior session ticker.
func ShouldResolveNewSymbolForProbe(msg string) bool {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return false
	}
	if isSymbolSwapUtterance(msg) {
		return true
	}
	if isStrategyOnlySwapUtterance(msg) {
		return false
	}
	q := ExtractStockQuery(msg)
	if q == "" {
		return false
	}
	return LooksLikeStockQuery(q) || StockQueryPlausible(q)
}

func isSymbolSwapUtterance(msg string) bool {
	return containsAny(msg, []string{
		"换一个标的", "换个标的", "换标的", "换一只", "换个股票",
		"换只股票", "换个标的试试", "换腾讯", "换苹果", "换茅台",
	})
}

func isStrategyOnlySwapUtterance(msg string) bool {
	return containsAny(msg, []string{
		"换一个策略", "换个策略", "换策略",
		"换一个信号", "换个信号", "换信号策略", "换一个信号策略",
		"换套策略", "换一套策略",
	})
}

func containsAny(msg string, tokens []string) bool {
	for _, tok := range tokens {
		if tok != "" && strings.Contains(msg, tok) {
			return true
		}
	}
	return false
}
