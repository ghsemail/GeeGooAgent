package strategydev

import (
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// regimeProbeBuySignal requests indicator series used for regime classification.
func regimeProbeBuySignal() []any {
	return []any{
		map[string]any{
			"index": "ADX", "type": "flag",
			"param": map[string]any{"period": "14"},
		},
		map[string]any{
			"index": "BBAND", "type": "flag",
			"param": map[string]any{"period": "20", "stdDev": "2", "matype": "0"},
		},
	}
}

func filterCompleteBars(bars []any, now time.Time) []any {
	if len(bars) == 0 {
		return bars
	}
	out := make([]any, 0, len(bars))
	for _, raw := range bars {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		vol := slots.FloatAny(row["volume"])
		if vol == 0 {
			continue
		}
		if ts := barTimeUnix(row); !ts.IsZero() && ts.After(now) {
			continue
		}
		out = append(out, row)
	}
	return out
}

func probeRules(data map[string]any) []map[string]any {
	raw, _ := data["buy_rules"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if row, ok := item.(map[string]any); ok {
			out = append(out, row)
		}
	}
	return out
}

func valueSeriesFromRules(rules []map[string]any) map[string][]float64 {
	out := map[string][]float64{}
	for _, rule := range rules {
		idx := strings.ToUpper(strings.TrimSpace(fmtIndex(rule["index"])))
		vs, _ := rule["value_series"].(map[string]any)
		if vs == nil {
			continue
		}
		for k, raw := range vs {
			vals := floatSlice(raw)
			if len(vals) == 0 {
				continue
			}
			key := strings.ToUpper(strings.TrimSpace(k))
			out[key] = vals
			if idx != "" {
				out[idx+"_"+key] = vals
			}
		}
	}
	return out
}

func fmtIndex(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		return strings.TrimSpace(fmt.Sprint(t["index"]))
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func floatSlice(raw any) []float64 {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]float64, len(arr))
	for i, v := range arr {
		out[i] = slots.FloatAny(v)
	}
	return out
}

func alignSeries(bars []any, series map[string][]float64) map[string][]float64 {
	n := len(bars)
	if n == 0 {
		return series
	}
	out := make(map[string][]float64, len(series))
	for k, vals := range series {
		if len(vals) >= n {
			out[k] = vals[len(vals)-n:]
		} else {
			out[k] = vals
		}
	}
	return out
}
