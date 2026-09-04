package slots

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestPickCombination(t *testing.T) {
	items := []map[string]any{
		{"name": "SAR信号配套MACD直方图趋势", "signal_id": "a", "buy_signal": []any{map[string]any{"index": "SAR"}}},
		{"name": "RSI阈值信号", "signal_id": "b", "buy_signal": []any{map[string]any{"index": "RSI"}}},
	}
	row, err := PickCombination(items, "SAR MACD")
	if err != nil {
		t.Fatal(err)
	}
	if row["signal_id"] != "a" {
		t.Fatalf("picked=%v", row["signal_id"])
	}
}

func TestResolveSignalCombination(t *testing.T) {
	runTool := func(_ context.Context, req tools.CallRequest, _ tools.Context) tools.Result {
		if req.Name != "get_signal_combinations" {
			t.Fatalf("unexpected tool %s", req.Name)
		}
		return tools.Result{
			Status: tools.StatusOK,
			Data: map[string]any{
				"items": []any{map[string]any{
					"name": "SAR信号配套MACD直方图趋势",
					"buy_signal": []any{map[string]any{"index": "SAR"}},
					"frequency":  "60m",
				}},
			},
		}
	}
	sig, err := ResolveSignal(context.Background(), tools.Context{}, runTool, SignalPlan{
		SignalQuery: "SAR MACD",
		SignalKind:  "combination",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sig.Buy) == 0 || sig.StrategyLabel == "" {
		t.Fatalf("sig=%+v", sig)
	}
}

func TestResolveSignalIndexClarify(t *testing.T) {
	items := []map[string]any{
		{"name": "RSI阈值信号", "index": "RSI", "frequency": "60m"},
		{"name": "RSI金死叉信号", "index": "RSICROSS", "frequency": "60m"},
	}
	runTool := func(_ context.Context, req tools.CallRequest, _ tools.Context) tools.Result {
		return tools.Result{Status: tools.StatusOK, Data: map[string]any{"items": []any{items[0], items[1]}}}
	}
	_, err := ResolveSignal(context.Background(), tools.Context{}, runTool, SignalPlan{
		SignalQuery: "RSI",
		SignalKind:  "indicator",
	})
	if err == nil {
		t.Fatal("expected clarify error without ClarifyFn")
	}
}

func TestApplySignalHeuristics(t *testing.T) {
	sp := SignalPlan{}
	ApplySignalHeuristics(&sp, "帮我回测 sar+macd 小米")
	if sp.SignalKind != "combination" || sp.SignalQuery != "SAR MACD" {
		t.Fatalf("sp=%+v", sp)
	}
}
