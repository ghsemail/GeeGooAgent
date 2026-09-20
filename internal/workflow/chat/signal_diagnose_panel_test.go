package chat

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestPhaseReadStrategySkipsCatalogWhenPanelOverrides(t *testing.T) {
	flow := &Flow{
		StrategyQuery: "4小时MACD市场节奏",
		Phase:         PhaseReadStrategy,
		ProbeBuyOverride: []any{
			map[string]any{"index": "SAR", "type": "signal"},
		},
		ProbeSellOverride: []any{
			map[string]any{"index": "MACD", "type": "flag"},
		},
		ProbePanelLabel: "买:Macd4H · 卖:SAR组合",
	}
	r := &Runner{
		RunTool: func(_ context.Context, req tools.CallRequest, _ tools.Context) tools.Result {
			t.Fatalf("catalog tools should not run with panel overrides, got %s", req.Name)
			return tools.Result{}
		},
	}
	if err := r.phaseSignalDiagnoseReadStrategy(context.Background(), flow, tools.Context{}, nil); err != nil {
		t.Fatal(err)
	}
	if flow.Phase != PhaseResolveSymbol {
		t.Fatalf("phase=%s", flow.Phase)
	}
	if flow.CatalogLabel != "买:Macd4H · 卖:SAR组合" {
		t.Fatalf("label=%q", flow.CatalogLabel)
	}
}

func TestResolveDiagnoseSignalFromPanelOverrides(t *testing.T) {
	flow := &Flow{
		ProbeBuyOverride: []any{map[string]any{"index": "SAR", "type": "signal"}},
		ProbeSellOverride: []any{
			map[string]any{"index": "MACD", "type": "flag"},
		},
		ProbeFrequencyOverride: "60m",
	}
	r := &Runner{}
	sig, err := r.resolveDiagnoseSignal(context.Background(), flow, tools.Context{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(sig.Buy) != 1 || len(sig.Sell) != 1 {
		t.Fatalf("buy=%d sell=%d", len(sig.Buy), len(sig.Sell))
	}
	if sig.Frequency != "60m" {
		t.Fatalf("freq=%s", sig.Frequency)
	}
	if sig.StrategyLabel != "买入: SAR · 卖出: MACD·flag" {
		t.Fatalf("label=%q", sig.StrategyLabel)
	}
}
