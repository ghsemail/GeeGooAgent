package strategydev

import (
	"encoding/json"
	"testing"
	"time"
)

func TestClassifyEMAStackLabel(t *testing.T) {
	if classifyEMAStackLabel(3, 2, 1, 2.5) != emaStackLabelBull {
		t.Fatal("expected bull")
	}
	if classifyEMAStackLabel(1, 2, 3, 1.5) != emaStackLabelBear {
		t.Fatal("expected bear")
	}
	if classifyEMAStackLabel(3, 2, 1, 1.5) != emaStackLabelChop {
		t.Fatal("price below mid should be chop")
	}
	if classifyEMAStackLabel(2, 3, 1, 2.5) != emaStackLabelChop {
		t.Fatal("non-monotonic EMA order should be chop")
	}
}

func TestRecursiveEMASeedFirstClose(t *testing.T) {
	closes := []float64{10, 11, 12}
	s25 := recursiveEMASeries(closes, 25)
	if len(s25) != 3 || s25[0] != 10 {
		t.Fatalf("seed=%v want first close 10", s25[0])
	}
	mult := 2.0 / 26.0
	want1 := (11-s25[0])*mult + s25[0]
	if s25[1] != want1 {
		t.Fatalf("ema[1]=%v want %v", s25[1], want1)
	}
}

func TestComputeEMAStacksAtAnchorTrendingUp(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	anchor := time.Date(2026, 6, 26, 16, 0, 0, 0, loc)
	makeBars := func(n int, step float64) []any {
		out := make([]any, n)
		for i := 0; i < n; i++ {
			c := 100.0 + float64(i)*step
			out[i] = map[string]any{
				"time": time.Date(2026, 1, 1, 0, 0, 0, 0, loc).AddDate(0, 0, i).Format("2006-01-02"),
				"close": c, "high": c + 1, "low": c - 1, "volume": 1.0,
			}
		}
		return out
	}
	barsByFreq := map[string][]any{
		"weekly": makeBars(130, 0.8),
		"daily":  makeBars(130, 0.5),
		"60m":    makeBars(130, 0.3),
	}
	fetchOK := map[string]bool{"weekly": true, "daily": true, "60m": true}
	stacks := ComputeEMAStacksAtAnchor(barsByFreq, fetchOK, nil, "60m", anchor)
	if stacks.Weekly.Label != emaStackLabelBull {
		t.Fatalf("weekly=%q want bull", stacks.Weekly.Label)
	}
	if stacks.Daily.Label != emaStackLabelBull {
		t.Fatalf("daily=%q want bull", stacks.Daily.Label)
	}
	if stacks.M60.Label != emaStackLabelBull {
		t.Fatalf("60m=%q want bull", stacks.M60.Label)
	}
}

func TestComputeEMAStacksInsufficientBarsUnknown(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	anchor := time.Date(2026, 6, 26, 16, 0, 0, 0, loc)
	short := make([]any, 40)
	for i := range short {
		short[i] = map[string]any{
			"time": time.Date(2026, 6, 1, 0, 0, 0, 0, loc).AddDate(0, 0, i).Format("2006-01-02"),
			"close": 100.0 + float64(i), "high": 101, "low": 99, "volume": 1.0,
		}
	}
	stacks := ComputeEMAStacksAtAnchor(map[string][]any{"daily": short}, map[string]bool{"daily": true}, nil, "60m", anchor)
	if stacks.Daily.Label != emaStackLabelUnknown {
		t.Fatalf("daily=%q want unknown", stacks.Daily.Label)
	}
}

func TestEpisodeTriggerRecordJSONIncludesDiagnoseFields(t *testing.T) {
	w := -0.5
	d := 0.5
	rec := EpisodeTriggerRecord{
		PriceScore:       -17.5,
		GatePriceScore:   -3.13,
		PAWeeklySubscore: &w,
		PADailySubscore:  &d,
		EMAStacks: &EMAStacks{
			Weekly: EMAStackEntry{Label: emaStackLabelBull, EMA25: 1, EMA50: 0.9, EMA120: 0.8, Close: 1.1},
			Daily:  EMAStackEntry{Label: emaStackLabelChop},
			M60:    EMAStackEntry{Label: emaStackLabelUnknown},
		},
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"gate_price_score", "pa_weekly_subscore", "pa_daily_subscore", "ema_stacks"} {
		if _, ok := m[key]; !ok {
			t.Fatalf("missing json key %q: %s", key, string(raw))
		}
	}
	stacks, _ := m["ema_stacks"].(map[string]any)
	if stacks["weekly"].(map[string]any)["label"] != emaStackLabelBull {
		t.Fatalf("weekly label missing: %v", stacks)
	}
	if _, ok := stacks["60m"]; !ok {
		t.Fatal("expected 60m key in ema_stacks")
	}
}
