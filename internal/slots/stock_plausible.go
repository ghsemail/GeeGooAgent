package slots

import "strings"

// StockQueryPlausible reports whether a plan stock_query looks like a ticker fragment,
// not a short intent phrase. Used to decide heuristic vs LLM plan extraction.
func StockQueryPlausible(q string) bool {
	q = strings.TrimSpace(q)
	if q == "" || !LooksLikeStockQuery(q) {
		return false
	}
	if reAShare.MatchString(q) || reHKCode.MatchString(q) {
		return true
	}
	upper := strings.ToUpper(q)
	if m := reUSTicker.FindStringSubmatch(upper); len(m) > 1 {
		if _, stop := tickerStopwords[m[1]]; !stop {
			return true
		}
	}
	for _, alias := range knownStockAliases {
		if q == alias {
			return true
		}
	}
	return len([]rune(q)) >= 3
}
