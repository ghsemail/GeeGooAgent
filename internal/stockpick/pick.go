package stockpick

import (
	"fmt"
	"strings"
)

// primaryStockAliases maps common Chinese nicknames to the primary listed ticker.
var primaryStockAliases = map[string]string{
	"腾讯":  "00700.HK",
	"小米":  "01810.HK",
	"茅台":  "600519.SH",
	"苹果":  "AAPL",
	"特斯拉": "TSLA",
	"英伟达": "NVDA",
	"微软":  "MSFT",
	"谷歌":  "GOOGL",
	"阿里":  "9988.HK",
	"拼多多": "PDD",
}

// AutoPick returns a single catalog row when the query clearly identifies one symbol.
func AutoPick(query string, items []map[string]any) (map[string]any, bool) {
	if len(items) == 0 {
		return nil, false
	}
	if len(items) == 1 {
		return items[0], true
	}
	if picked, ok := pickByCode(items, query); ok {
		return picked, true
	}
	if picked, ok := pickByPrimaryAlias(query, items); ok {
		return picked, true
	}
	if picked, ok := pickByNameRank(query, items); ok {
		return picked, true
	}
	return nil, false
}

func pickByCode(items []map[string]any, query string) (map[string]any, bool) {
	q := strings.ToUpper(strings.TrimSpace(query))
	if q == "" || len(items) == 0 {
		return nil, false
	}
	qDigits := strings.TrimLeft(q, "0")
	var containsMatches []map[string]any
	for _, row := range items {
		code := strings.ToUpper(strings.TrimSpace(fmt.Sprint(row["code"])))
		if code == "" {
			continue
		}
		if code == q {
			return row, true
		}
		codeBase := strings.TrimSuffix(code, ".HK")
		codeBase = strings.TrimSuffix(codeBase, ".SH")
		codeBase = strings.TrimSuffix(codeBase, ".SZ")
		codeBase = strings.TrimSuffix(codeBase, ".US")
		if strings.HasSuffix(q, ".HK") || strings.HasSuffix(q, ".SH") || strings.HasSuffix(q, ".SZ") || strings.HasSuffix(q, ".US") {
			if code == q || strings.HasPrefix(code, q) {
				return row, true
			}
		}
		if qDigits != "" && (codeBase == qDigits || strings.HasSuffix(codeBase, qDigits) || strings.HasSuffix(qDigits, strings.TrimLeft(codeBase, "0"))) {
			return row, true
		}
		if strings.Contains(code, q) || strings.Contains(q, codeBase) {
			containsMatches = append(containsMatches, row)
		}
	}
	if len(containsMatches) == 1 {
		return containsMatches[0], true
	}
	return nil, false
}

func pickByPrimaryAlias(query string, items []map[string]any) (map[string]any, bool) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, false
	}
	wantCode, ok := primaryStockAliases[q]
	if !ok {
		return nil, false
	}
	wantCode = strings.ToUpper(wantCode)
	for _, row := range items {
		code := strings.ToUpper(strings.TrimSpace(fmt.Sprint(row["code"])))
		if code == wantCode {
			return row, true
		}
	}
	return nil, false
}

func pickByNameRank(query string, items []map[string]any) (map[string]any, bool) {
	q := strings.TrimSpace(query)
	if q == "" || len(items) < 2 {
		return nil, false
	}
	type scored struct {
		row   map[string]any
		score int
	}
	var ranked []scored
	for _, row := range items {
		name := strings.TrimSpace(fmt.Sprint(row["name"]))
		if name == "" {
			continue
		}
		score := nameMatchScore(q, name)
		if score <= 0 {
			continue
		}
		ranked = append(ranked, scored{row: row, score: score})
	}
	if len(ranked) == 0 {
		return nil, false
	}
	best := ranked[0]
	for _, cand := range ranked[1:] {
		if cand.score > best.score {
			best = cand
		}
	}
	if len(ranked) > 1 {
		secondScore := -1
		bestCode := rowCode(best.row)
		for _, cand := range ranked {
			if rowCode(cand.row) == bestCode {
				continue
			}
			if cand.score > secondScore {
				secondScore = cand.score
			}
		}
		if secondScore >= 0 && best.score-secondScore < 2 {
			return nil, false
		}
	}
	return best.row, true
}

func nameMatchScore(query, name string) int {
	q := strings.TrimSpace(query)
	n := strings.TrimSpace(name)
	if q == "" || n == "" {
		return 0
	}
	if n == q {
		return 100
	}
	if strings.HasPrefix(n, q) {
		return 80 - minInt(len([]rune(n))-len([]rune(q)), 20)
	}
	if strings.Contains(n, q) {
		penalty := minInt(len([]rune(n))-len([]rune(q)), 30)
		return 50 - penalty
	}
	return 0
}

func rowCode(row map[string]any) string {
	return strings.ToUpper(strings.TrimSpace(fmt.Sprint(row["code"])))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
