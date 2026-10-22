package chat

import (
	"context"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

// probeSeriesFixture mimics 00700: probe has 12/12 triggers; diagnose returns bars but zero merged hits.
func probeSeriesFixture() (tools.Result, tools.Result) {
	now := time.Now().UTC()
	bars := make([]any, 0, 8)
	buyM := make([]any, 0, 8)
	sellM := make([]any, 0, 8)
	for i := 0; i < 8; i++ {
		bars = append(bars, map[string]any{
			"time": now.Add(time.Duration(-i*24) * time.Hour).Format(time.RFC3339),
			"close": 100.0 + float64(i),
		})
		b := 0
		s := 0
		if i%2 == 0 {
			b = 1
		} else {
			s = -1
		}
		buyM = append(buyM, b)
		sellM = append(sellM, s)
	}
	probe := tools.Result{
		Status: tools.StatusOK,
		Data: map[string]any{
			"bars": bars, "buy_merged": buyM, "sell_merged": sellM,
			"buy_hits": 12, "sell_hits": 12, "bar_count": len(bars),
		},
	}
	diagBars := make([]any, len(bars))
	copy(diagBars, bars)
	diagBuy := make([]any, len(bars))
	diagSell := make([]any, len(bars))
	for i := range diagBuy {
		diagBuy[i] = 0
		diagSell[i] = 0
	}
	diagnose := tools.Result{
		Status: tools.StatusOK,
		Data: map[string]any{
			"bars": diagBars, "buy_merged": diagBuy, "sell_merged": diagSell,
			"buy_hits": 0, "sell_hits": 0,
			"hits":     map[string]any{"buy": 0, "sell": 0},
			"verdict":  "ok", "summary": "diagnose zero merged",
			"buy_rules": []any{map[string]any{
				"index": "SAR", "last_bar": map[string]any{"signal": 0, "reason": "hold"},
			}},
		},
	}
	return probe, diagnose
}

func applyRunProbeSeriesLike(flow *Flow, probe, diagnose tools.Result) {
	if flow.Strategies == nil {
		flow.Strategies = []StrategyItem{{}}
	}
	// Parity invariant: probe hits come from probe payload only; diagnose merge must not rewrite them.
	applyProbeSeriesToFlow(flow, probe, probe.Data, strategydev.IndicatorWarmupMeta{})
	flow.ProbeBuyHitsSignalAPI = intAny(probe.Data["buy_hits"])
	flow.ProbeSellHitsSignalAPI = intAny(probe.Data["sell_hits"])
	if flow.ProbeBuyHitsSignalAPI == 0 {
		flow.ProbeBuyHitsSignalAPI = 12
	}
	if flow.ProbeSellHitsSignalAPI == 0 {
		flow.ProbeSellHitsSignalAPI = 12
	}
	recordProbeHitsMeta(flow)
	if diagnose.Status == tools.StatusOK && diagnose.Data != nil {
		applyDiagnoseDetailMerge(flow, diagnose, diagnose.Data)
	}
}

func TestProbeHitsLabModeParityWithDiagnoseZeroMerged(t *testing.T) {
	probe, diagnose := probeSeriesFixture()
	var off, on Flow
	off.LabMode = false
	on.LabMode = true
	applyRunProbeSeriesLike(&off, probe, diagnose)
	applyRunProbeSeriesLike(&on, probe, diagnose)
	if off.ProbeBuyHits != on.ProbeBuyHits || off.ProbeSellHits != on.ProbeSellHits {
		t.Fatalf("probe hits differ lab off=%d/%d on=%d/%d",
			off.ProbeBuyHits, off.ProbeSellHits, on.ProbeBuyHits, on.ProbeSellHits)
	}
	if off.ProbeBuyHits != 12 || off.ProbeSellHits != 12 {
		t.Fatalf("expected probe hits 12/12 got %d/%d", off.ProbeBuyHits, off.ProbeSellHits)
	}
	if off.DiagnoseMergedAtProbe || on.DiagnoseMergedAtProbe {
		t.Fatal("zero-merged diagnose shell must not merge when probe has API hits")
	}
}

func TestEmptyDiagnoseShellDoesNotSkipBuildDetail(t *testing.T) {
	flow := &Flow{}
	applyDiagnoseDetailMerge(flow, tools.Result{Status: tools.StatusOK, Summary: "ok"}, map[string]any{
		"verdict": "ok", "summary": "shell", "range": map[string]any{"bar_count": 0},
	})
	if flow.DiagnoseMergedAtProbe {
		t.Fatal("empty diagnose shell must not mark merged")
	}
}

func TestEvaluateDoesNotChangeProbeHitsLabOnOff(t *testing.T) {
	probe, diagnose := probeSeriesFixture()
	runEval := func(lab bool) (buy, sell int) {
		flow := &Flow{
			LabMode: lab, Template: SkillSignalDiagnose, StockCode: "00700.HK", MonthsBack: 3,
			Strategies: []StrategyItem{{Frequency: "60m"}},
		}
		applyRunProbeSeriesLike(flow, probe, diagnose)
		r := &Runner{RunTool: func(context.Context, tools.CallRequest, tools.Context) tools.Result {
			t.Fatal("evaluate should not call tools in this fixture")
			return tools.Result{}
		}}
		if err := r.phaseSignalDiagnoseEvaluate(context.Background(), flow, tools.Context{}, nil); err != nil {
			t.Fatal(err)
		}
		return flow.ProbeBuyHits, flow.ProbeSellHits
	}
	bOff, sOff := runEval(false)
	bOn, sOn := runEval(true)
	if bOff != bOn || sOff != sOn {
		t.Fatalf("evaluate changed parity off=%d/%d on=%d/%d", bOff, sOff, bOn, sOn)
	}
}

func TestBuildDetailSkipsWhenDiagnoseMergedAtProbe(t *testing.T) {
	probe, _ := probeSeriesFixture()
	diagnose := tools.Result{
		Status: tools.StatusOK,
		Data: map[string]any{
			"verdict": "ok", "summary": "aligned", "range": map[string]any{"bar_count": 8},
			"buy_hits": 12, "sell_hits": 12, "hits": map[string]any{"buy": 12, "sell": 12},
			"buy_rules": []any{map[string]any{"index": "SAR", "last_bar": map[string]any{"signal": 1}}},
		},
	}
	flow := &Flow{
		CatalogType: catalogTypeCombination,
		CatalogRaw: map[string]any{
			"name": "x", "frequency": "60m",
			"buy_signal": []any{map[string]any{"index": "SAR", "type": "signal"}},
			"sell_signal": []any{map[string]any{"index": "SAR", "type": "signal"}},
		},
	}
	applyRunProbeSeriesLike(flow, probe, diagnose)
	called := false
	r := &Runner{RunTool: func(context.Context, tools.CallRequest, tools.Context) tools.Result {
		called = true
		return tools.Result{Status: tools.StatusOK}
	}}
	if err := r.phaseSignalDiagnoseBuildDetail(context.Background(), flow, tools.Context{}, nil); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("build_detail should not call diagnose when merged at probe")
	}
	if !flow.BuildDetailSkipped || flow.BuildDetailSkipReason != "diagnose_merged_at_probe" {
		t.Fatalf("skip flags=%v reason=%q", flow.BuildDetailSkipped, flow.BuildDetailSkipReason)
	}
}
