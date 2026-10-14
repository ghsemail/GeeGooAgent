package strategydev

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestFetchPriceAnalysisBarsProbeArgsNoLimit(t *testing.T) {
	var captured map[string]any
	run := ToolRunner(func(ctx context.Context, name string, args map[string]any) tools.Result {
		if name != "probe_bot_signal_series" {
			t.Fatalf("tool=%s", name)
		}
		captured = args
		return tools.Result{Status: tools.StatusOK, Data: map[string]any{"bars": []any{}}}
	})
	_, ok := fetchPriceAnalysisBarsWithAsOf(context.Background(), run, "00700.HK", "60m", 3, nil)
	if !ok {
		t.Fatal("expected fetch ok")
	}
	if captured == nil {
		t.Fatal("no probe call")
	}
	if _, hasLimit := captured["limit"]; hasLimit {
		t.Fatalf("must not pass limit: %v", captured)
	}
	if captured["months_back"] != 3 {
		t.Fatalf("months_back=%v", captured["months_back"])
	}
}

func TestFetchPriceAnalysisBarsRetriesTransientFailure(t *testing.T) {
	attempts := 0
	run := ToolRunner(func(ctx context.Context, name string, args map[string]any) tools.Result {
		attempts++
		if attempts < 2 {
			return tools.Result{Status: tools.StatusError, Summary: "cold start timeout"}
		}
		return tools.Result{Status: tools.StatusOK, Data: map[string]any{
			"bars": []any{map[string]any{"close": 1.0, "high": 2, "low": 0.5, "volume": 100.0}},
		}}
	})
	bars, ok := fetchPriceAnalysisBarsWithAsOf(context.Background(), run, "00700.HK", "5m", 3, nil)
	if !ok || len(bars) != 1 {
		t.Fatalf("ok=%v bars=%d attempts=%d", ok, len(bars), attempts)
	}
	if attempts != 2 {
		t.Fatalf("attempts=%d want 2", attempts)
	}
}

func TestFetchPriceAnalysisBarsProbeCacheReusesWithinContext(t *testing.T) {
	calls := 0
	run := ToolRunner(func(ctx context.Context, name string, args map[string]any) tools.Result {
		calls++
		return tools.Result{Status: tools.StatusOK, Data: map[string]any{
			"bars": []any{map[string]any{"close": 1.0, "high": 2, "low": 0.5, "volume": 100.0}},
		}}
	})
	ctx := ContextWithPriceProbeCache(context.Background())
	_, ok1 := fetchPriceAnalysisBarsWithAsOf(ctx, run, "AAPL", "60m", 3, nil)
	_, ok2 := fetchPriceAnalysisBarsWithAsOf(ctx, run, "AAPL", "60m", 3, nil)
	if !ok1 || !ok2 || calls != 1 {
		t.Fatalf("ok1=%v ok2=%v calls=%d", ok1, ok2, calls)
	}
}

func TestFetchPriceAnalysisBarsFetchFailed(t *testing.T) {
	attempts := 0
	run := ToolRunner(func(ctx context.Context, name string, args map[string]any) tools.Result {
		attempts++
		return tools.Result{Status: tools.StatusError, Summary: "timeout"}
	})
	bars, ok := fetchPriceAnalysisBarsWithAsOf(context.Background(), run, "00700.HK", "daily", 3, nil)
	if ok || len(bars) != 0 {
		t.Fatalf("ok=%v bars=%d", ok, len(bars))
	}
	if attempts != priceAnalysisProbeMaxAttempts {
		t.Fatalf("attempts=%d want %d", attempts, priceAnalysisProbeMaxAttempts)
	}
}
