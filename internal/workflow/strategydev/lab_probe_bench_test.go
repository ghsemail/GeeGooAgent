package strategydev

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// BenchmarkMultiTFPriceProbeParallel measures concurrent multi-TF probe scheduling (mock I/O).
func BenchmarkMultiTFPriceProbeParallel(b *testing.B) {
	ConfigurePriceAnalysisEnabled(true)
	b.Cleanup(func() { ConfigurePriceAnalysisEnabled(true) })
	var calls atomic.Int32
	run := ToolRunner(func(ctx context.Context, name string, args map[string]any) tools.Result {
		calls.Add(1)
		time.Sleep(2 * time.Millisecond)
		return tools.Result{Status: tools.StatusOK, Data: map[string]any{"bars": []any{map[string]any{"time": "t", "close": 1.0}}}}
	})
	in := MultiTimeframePriceBarsInput{Run: run, Code: "00700.HK", Months: 3, GateFreq: "60m"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FetchMultiTimeframePriceAnalysisBarsParallel(context.Background(), in)
	}
	b.ReportMetric(float64(calls.Load())/float64(b.N), "probes/op")
}

func TestMultiTFPriceProbeParallelFasterThanSerialBudget(t *testing.T) {
	ConfigurePriceAnalysisEnabled(true)
	delay := 8 * time.Millisecond
	run := ToolRunner(func(ctx context.Context, name string, args map[string]any) tools.Result {
		time.Sleep(delay)
		return tools.Result{Status: tools.StatusOK, Data: map[string]any{"bars": []any{map[string]any{"time": "t", "close": 1.0}}}}
	})
	in := MultiTimeframePriceBarsInput{Run: run, Code: "AAPL.US", Months: 3, GateFreq: "60m"}
	start := time.Now()
	FetchMultiTimeframePriceAnalysisBarsParallel(context.Background(), in)
	parallel := time.Since(start)
	serialBudget := delay * 4
	if parallel > serialBudget {
		t.Fatalf("parallel fetch too slow: %v > serial budget %v", parallel, serialBudget)
	}
	t.Logf("parallel multi-TF mock fetch=%v (budget<%v)", parallel, serialBudget)
}

func ExampleLabDetailRef() {
	fmt.Println(LabDetailRef("buy", 2))
	// Output: buy:2
}
