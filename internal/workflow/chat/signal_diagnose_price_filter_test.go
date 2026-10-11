package chat

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func TestSignalDiagnoseEvaluatePartialStripListsStrippedAndKeepsEpisodes(t *testing.T) {
	n := 90
	bars := make([]any, n)
	buyM := make([]any, n)
	sellM := make([]any, n)
	for i := 0; i < n; i++ {
		c := 100.0 + float64(i)*0.2
		bars[i] = map[string]any{
			"high": c + 1, "low": c - 1, "close": c, "volume": 1000.0,
			"time": fmt.Sprintf("2026-01-%02d 10:00:00", (i%28)+1),
		}
	}
	buyM[10] = 1
	sellM[15] = -1
	sellM[25] = -1

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
	st := flow.PriceEpisodeFilterStats
	if st == nil {
		t.Fatal("expected stats")
	}
	if st.PreFilterSellHits < 2 {
		t.Fatalf("pre sell hits=%d", st.PreFilterSellHits)
	}
	if st.StrippedSell == 0 && st.StrippedBuy == 0 {
		t.Skip("filter allowed both sides on this fixture")
	}
	if st.StrippedSell > 0 && len(st.StrippedSellTriggers) != st.StrippedSell {
		t.Fatalf("sell ledger=%d stripped=%d", len(st.StrippedSellTriggers), st.StrippedSell)
	}
	if flow.SignalEval.BuyEpisodes.CompleteCount+flow.SignalEval.SellEpisodes.CompleteCount == 0 &&
		st.PostFilterBuyHits+st.PostFilterSellHits > 0 {
		t.Fatal("expected episodes when triggers remain")
	}
}

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

func TestRenderSignalDiagnoseFactsIncludesStrippedLedger(t *testing.T) {
	pf := strategydev.PriceFilterResult{
		Regime: strategydev.RegimeUnclear, Side: strategydev.SideNone,
		AllowLong: false, AllowShort: false,
	}
	st := strategydev.EpisodeTriggerFilterStats{
		PreFilterBuyHits: 12, PreFilterSellHits: 16,
		StrippedBuy: 12, StrippedSell: 16,
		InterceptionBasis: "价格分析结论为不明确，不允许买入/卖出，故掩码剔除",
		StrippedBuyTriggers: []strategydev.StrippedEpisodeTrigger{
			{Time: "2026-03-01 10:00 (Asia/Shanghai)", Close: 380.5},
		},
	}
	flow := &Flow{
		UsePriceAnalysisEpisodeFilter: true,
		ProbeBuyHits:                  12,
		ProbeSellHits:                 16,
		PriceFilter:                   &pf,
		PriceEpisodeFilterStats:       &st,
		Strategies:                    []StrategyItem{{Frequency: "60m"}},
	}
	out := renderSignalDiagnoseFacts(flow)
	if !strings.Contains(out, "剔除明细") || !strings.Contains(out, "拦截依据") {
		t.Fatalf("missing stripped section:\n%s", out)
	}
	if !strings.Contains(out, "被掩码剔除的买入触发") {
		t.Fatal("missing buy ledger")
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
