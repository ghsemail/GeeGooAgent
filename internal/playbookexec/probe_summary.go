package playbookexec

import (
	"fmt"
	"strings"
)

type probeSummary struct {
	Code            string
	Frequency       string
	MonthsBack      int
	BuyHits         int
	SellHits        int
	BarCount        int
	RangeStart      string
	RangeEnd        string
	RecentBuys      []string
	RecentSells     []string
	RecentSignals   []map[string]any
}

func extractProbeSummary(data map[string]any) probeSummary {
	out := probeSummary{}
	if data == nil {
		return out
	}
	out.Code = strings.TrimSpace(fmt.Sprint(data["code"]))
	out.Frequency = strings.TrimSpace(fmt.Sprint(data["frequency"]))
	if mb, ok := data["months_back"].(float64); ok {
		out.MonthsBack = int(mb)
	}
	bars, _ := data["bars"].([]any)
	out.BarCount = len(bars)
	if len(bars) > 0 {
		if first, ok := bars[0].(map[string]any); ok {
			out.RangeStart = fmt.Sprint(first["time"])
		}
		if last, ok := bars[len(bars)-1].(map[string]any); ok {
			out.RangeEnd = fmt.Sprint(last["time"])
		}
	}
	out.BuyHits = countMergedSignals(data["buy_merged"], 1)
	out.SellHits = countMergedSignals(data["sell_merged"], -1)
	out.RecentBuys = recentSignalTimes(bars, data["buy_merged"], 1, 5)
	out.RecentSells = recentSignalTimes(bars, data["sell_merged"], -1, 5)
	out.RecentSignals = mergeRecentSignals(bars, data["buy_merged"], data["sell_merged"], 12)
	return out
}

func countMergedSignals(raw any, target int) int {
	arr, ok := raw.([]any)
	if !ok {
		return 0
	}
	n := 0
	for _, item := range arr {
		switch v := item.(type) {
		case float64:
			if int(v) == target {
				n++
			}
		case int:
			if v == target {
				n++
			}
		}
	}
	return n
}

func recentSignalTimes(bars []any, mergedRaw any, target, maxN int) []string {
	merged, ok := mergedRaw.([]any)
	if !ok || len(merged) == 0 || maxN <= 0 {
		return nil
	}
	limit := len(merged)
	if len(bars) < limit {
		limit = len(bars)
	}
	out := make([]string, 0, maxN)
	for i := limit - 1; i >= 0 && len(out) < maxN; i-- {
		if !mergedHit(merged[i], target) {
			continue
		}
		bar, ok := bars[i].(map[string]any)
		if !ok {
			continue
		}
		if ts := strings.TrimSpace(fmt.Sprint(bar["time"])); ts != "" {
			out = append(out, ts)
		}
	}
	return out
}

func mergeRecentSignals(bars []any, buyMerged, sellMerged any, max int) []map[string]any {
	buys, _ := buyMerged.([]any)
	sells, _ := sellMerged.([]any)
	if len(bars) == 0 {
		return nil
	}
	limit := len(bars)
	if len(buys) > 0 && len(buys) < limit {
		limit = len(buys)
	}
	if len(sells) > 0 && len(sells) < limit {
		limit = len(sells)
	}
	type item struct {
		time string
		row  map[string]any
	}
	var items []item
	for i := 0; i < limit; i++ {
		bar, ok := bars[i].(map[string]any)
		if !ok {
			continue
		}
		ts := strings.TrimSpace(fmt.Sprint(bar["time"]))
		if ts == "" {
			continue
		}
		if len(buys) > i && mergedHit(buys[i], 1) {
			items = append(items, item{time: ts, row: map[string]any{
				"time": ts, "direction": "买入", "close": bar["close"],
			}})
		}
		if len(sells) > i && mergedHit(sells[i], -1) {
			items = append(items, item{time: ts, row: map[string]any{
				"time": ts, "direction": "卖出", "close": bar["close"],
			}})
		}
	}
	if len(items) == 0 {
		return nil
	}
	// keep last max entries
	if len(items) > max {
		items = items[len(items)-max:]
	}
	out := make([]map[string]any, len(items))
	for i, it := range items {
		out[i] = it.row
	}
	return out
}

func mergedHit(raw any, target int) bool {
	switch v := raw.(type) {
	case float64:
		return int(v) == target
	case int:
		return v == target
	default:
		return false
	}
}
