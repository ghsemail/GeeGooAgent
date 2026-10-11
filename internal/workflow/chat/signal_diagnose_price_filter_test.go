package chat

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestSignalDiagnoseEvaluateAppliesPriceFilterWhenEnabled(t *testing.T) {
	n := 90
	bars := make([]any, n)
	buyM := make([]any, n)
	sellM := make([]any, n)
	for i := 0; i < n; i++ {
		c := 100.0 + float64(i%5)
		bars[i] = map[string]any{"high": c + 1, "low": c - 1, "close": c, "volume": 1000.0}
	}
	buyM[10] = 1
	buyM[20] = 1
	sellM[15] = -1

	flow := &Flow{
		UsePriceAnalysisEpisodeFilter: true,
		Strategies:                    []StrategyItem{{Frequency: "60m"}},
		ProbeRaw: map[string]any{
			"bars": bars, "buy_merged": buyM, "sell_merged": sellM,
		},
	}
	r := &Runner{}
	if err := r.phaseSignalDiagnoseEvaluate(context.Background(), flow); err != nil {
		t.Fatal(err)
	}
	if flow.PriceFilter == nil {
		t.Fatal("expected price filter snapshot")
	}
}

func TestSignalDiagnoseEvaluateSkipsPriceFilterWhenDisabled(t *testing.T) {
	n := 90
	bars := make([]any, n)
	buyM := make([]any, n)
	sellM := make([]any, n)
	for i := 0; i < n; i++ {
		c := 100.0
		bars[i] = map[string]any{"high": c + 1, "low": c - 1, "close": c, "volume": 1000.0}
	}
	buyM[5] = 1
	flow := &Flow{
		UsePriceAnalysisEpisodeFilter: false,
		Strategies:                    []StrategyItem{{Frequency: "60m"}},
		ProbeRaw: map[string]any{
			"bars": bars, "buy_merged": buyM, "sell_merged": sellM,
		},
	}
	before := slots.EvaluateSignalEpisodes(bars, buyM, sellM)
	r := &Runner{}
	if err := r.phaseSignalDiagnoseEvaluate(context.Background(), flow); err != nil {
		t.Fatal(err)
	}
	if flow.PriceFilter != nil {
		t.Fatal("filter should be skipped")
	}
	if len(flow.SignalEval.BuyDetails) != len(before.BuyDetails) {
		t.Fatalf("buy episodes changed when filter off")
	}
}

func TestParsePriceAnalysisEpisodeFilterOpt(t *testing.T) {
	raw := map[string]any{
		"signal_diagnose": map[string]any{
			"use_price_analysis_episode_filter": false,
		},
	}
	o := ParseSignalDiagnoseOptsFromWorkflowOptions(raw)
	if o == nil || o.UsePriceAnalysisEpisodeFilter == nil || *o.UsePriceAnalysisEpisodeFilter {
		t.Fatal("expected explicit false")
	}
	session := runtime.NewSession()
	session.PendingSignalDiagnoseOpts = o
	flow := newSignalDiagnoseFlow("诊断 SAR · 00700.HK")
	ApplyPendingSignalDiagnoseOpts(session, flow)
	if flow.UsePriceAnalysisEpisodeFilter {
		t.Fatal("pending opts should disable filter")
	}
}
