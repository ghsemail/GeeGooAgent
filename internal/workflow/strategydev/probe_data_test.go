package strategydev

import (
	"testing"
	"time"
)

func TestBarTimeUnixNaiveStringUsesAsiaShanghai(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	row := map[string]any{"time": "2026-06-29 12:00:00"}
	ts := barTimeUnix(row)
	want := time.Date(2026, 6, 29, 12, 0, 0, 0, loc)
	if !ts.Equal(want) {
		t.Fatalf("got %v want %v", ts.In(loc), want)
	}
}

func TestBarTimeUnixCNCompactWithSuffix(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	row := map[string]any{"time": "202604291130 00:00:00"}
	ts := barTimeUnix(row)
	want := time.Date(2026, 4, 29, 11, 30, 0, 0, loc)
	if !ts.Equal(want) {
		t.Fatalf("got %v want %v", ts.In(loc), want)
	}
}

func TestBarTimeUnixDateOnlyUsesAsiaShanghaiMidnight(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	row := map[string]any{"time": "2026-06-26"}
	ts := barTimeUnix(row)
	want := time.Date(2026, 6, 26, 0, 0, 0, 0, loc)
	if !ts.Equal(want) {
		t.Fatalf("got %v want %v", ts.In(loc), want)
	}
}
