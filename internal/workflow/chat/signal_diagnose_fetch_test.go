package chat

import (
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func TestApplySignalDiagnoseSeriesDataProbeOnly(t *testing.T) {
	flow := &Flow{Strategies: []StrategyItem{{}}}
	data := map[string]any{
		"buy_hits": 2, "sell_hits": 1,
		"bars": []any{
			map[string]any{"time": time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339), "close": 1.0},
			map[string]any{"time": time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339), "close": 2.0},
		},
		"buy_merged":  []any{1, 0},
		"sell_merged": []any{0, -1},
	}
	res := tools.Result{Status: tools.StatusOK, Data: data}
	applySignalDiagnoseSeriesData(flow, res, 1, nil, nil, false)
	if flow.DiagnoseMergedAtProbe {
		t.Fatal("probe-only apply must not set merged flag")
	}
	if flow.ProbeRaw == nil || flow.ProbeRaw["bars"] == nil {
		t.Fatal("expected probe raw bars")
	}
}

func TestApplyDiagnoseMergeDoesNotOverwriteProbeHits(t *testing.T) {
	probe, diagnose := probeSeriesFixture()
	flow := &Flow{Strategies: []StrategyItem{{}}}
	applyProbeSeriesToFlow(flow, probe, probe.Data, strategydev.IndicatorWarmupMeta{})
	beforeBuy, beforeSell := flow.ProbeBuyHits, flow.ProbeSellHits
	applyDiagnoseDetailMerge(flow, diagnose, diagnose.Data)
	if flow.ProbeBuyHits != beforeBuy || flow.ProbeSellHits != beforeSell {
		t.Fatalf("diagnose merge overwrote hits: before %d/%d after %d/%d",
			beforeBuy, beforeSell, flow.ProbeBuyHits, flow.ProbeSellHits)
	}
	if !flow.DiagnoseMergedAtProbe {
		t.Fatal("expected merged flag")
	}
}
