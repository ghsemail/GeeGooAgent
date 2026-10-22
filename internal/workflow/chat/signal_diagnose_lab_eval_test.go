package chat

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func TestLabModeSkipsPriceEpisodeFilter(t *testing.T) {
	strategydev.SetPriceAnalysisFeatureEnabledForTest(t, true)
	bars := []any{
		map[string]any{"time": "t0", "close": 100.0, "high": 101.0, "low": 99.0},
		map[string]any{"time": "t1", "close": 101.0, "high": 102.0, "low": 100.0},
	}
	buyMerged := []any{0, 1}
	sellMerged := []any{0, -1}
	flow := &Flow{
		Template: SkillSignalDiagnose, LabMode: true, StockCode: "00700.HK",
		ProbeRaw: map[string]any{
			"bars": bars, "buy_merged": buyMerged, "sell_merged": sellMerged,
		},
		MonthsBack: 3,
	}
	r := &Runner{
		RunTool: func(ctx context.Context, req tools.CallRequest, toolCtx tools.Context) tools.Result {
			t.Fatal("price filter must not invoke tools when lab_mode")
			return tools.Result{}
		},
	}
	if err := r.phaseSignalDiagnoseEvaluate(context.Background(), flow, tools.Context{}, nil); err != nil {
		t.Fatal(err)
	}
	if len(flow.SignalEval.BuyDetails) == 0 && len(flow.SignalEval.SellDetails) == 0 {
		t.Fatal("expected episodes without price gate")
	}
	if flow.PriceEpisodeFilterStats != nil {
		t.Fatal("lab mode should not populate price episode filter stats")
	}
	if len(flow.LabEpisodeSummaries) == 0 {
		t.Fatal("expected lab episode summaries")
	}
	_ = slots.EvaluateSignalEpisodes
}
