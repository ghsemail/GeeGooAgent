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
	_, ok := fetchPriceAnalysisBars(context.Background(), run, "00700.HK", "60m", 3)
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

func TestFetchPriceAnalysisBarsFetchFailed(t *testing.T) {
	run := ToolRunner(func(ctx context.Context, name string, args map[string]any) tools.Result {
		return tools.Result{Status: tools.StatusError, Summary: "timeout"}
	})
	bars, ok := fetchPriceAnalysisBars(context.Background(), run, "00700.HK", "daily", 3)
	if ok || len(bars) != 0 {
		t.Fatalf("ok=%v bars=%d", ok, len(bars))
	}
}
