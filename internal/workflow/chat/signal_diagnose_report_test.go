package chat

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestRenderSignalDiagnoseReportCompactNotKBDraft(t *testing.T) {
	flow := flowWithFullEval()
	flow.EvalJudgment = "结论：样本内买点有效，注意卖段稀疏。"
	flow.KBDraft = assembleSignalDiagnoseDoc(flow, flow.EvalJudgment)
	flow.KnowledgeID = "kb-1"
	flow.KnowledgeTitle = "SAR · 腾讯控股 · 信号诊断"
	report := renderSignalDiagnoseReport(flow)
	if strings.Contains(report, flow.KBDraft) {
		t.Fatal("chat report must not echo full KB draft")
	}
	for _, forbidden := range []string{
		"命中率与有效性怎么算",
		"汇总 · Strict",
		"持有K线",
		"买入规则",
		"样例触发",
		"买入 Episode 明细",
		"### 一、信号触发情况",
		"Strict / Path 命中率（摘要",
	} {
		if strings.Contains(report, forbidden) {
			t.Fatalf("chat report must omit %q:\n%s", forbidden, report)
		}
	}
	for _, want := range []string{
		"### 波段盈亏（Strict 闭合",
		"#### 买入（做多）",
		"#### 做空（卖出）",
		"**总计波段**",
		"**盈利波段**",
		"**累计涨幅**",
		"**亏损波段**",
		"**累计亏损**",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("chat report missing wave card field %q:\n%s", want, report)
		}
	}
	if !strings.Contains(report, "kb-1") {
		t.Fatalf("missing knowledge id:\n%s", report)
	}
	if len(report) >= len(flow.KBDraft) {
		t.Fatalf("expected compact report (%d) << kb draft (%d)", len(report), len(flow.KBDraft))
	}
}

func TestComputeStrictWavePnLFromEpisodes(t *testing.T) {
	details := []slots.SignalEpisodeDetail{
		{Complete: true, Hit: true, DirectionReturn: 0.10},
		{Complete: true, Hit: true, DirectionReturn: 0.05},
		{Complete: true, Hit: false, DirectionReturn: -0.03},
		{Complete: true, Hit: false, DirectionReturn: 0},
		{Complete: false, Hit: true, DirectionReturn: 0.99},
	}
	got := computeStrictWavePnL(details)
	if got.Total != 4 || got.WinCount != 2 || got.LossCount != 2 {
		t.Fatalf("counts: %+v", got)
	}
	if got.WinPctSum < 0.1499 || got.WinPctSum > 0.1501 {
		t.Fatalf("win sum want ~0.15 got %v", got.WinPctSum)
	}
	if got.LossPctSum < 0.0299 || got.LossPctSum > 0.0301 {
		t.Fatalf("loss sum want ~0.03 got %v", got.LossPctSum)
	}
}

func TestRenderSignalDiagnoseReportWaveNumbersBuySell(t *testing.T) {
	flow := flowWithFullEval()
	flow.SignalEval = slots.SignalEpisodeEval{
		Method: "until_opposite_signal",
		BuyDetails: []slots.SignalEpisodeDetail{
			{Complete: true, Hit: true, DirectionReturn: 0.08},
			{Complete: true, Hit: true, DirectionReturn: 0.02},
			{Complete: true, Hit: false, DirectionReturn: -0.01},
		},
		SellDetails: []slots.SignalEpisodeDetail{
			{Complete: true, Hit: true, DirectionReturn: 0.04},
			{Complete: true, Hit: false, DirectionReturn: -0.06},
		},
		BuyEpisodes:  slots.SignalEpisodeMetrics{CompleteCount: 3, HitCount: 2, IncompleteCount: 0},
		SellEpisodes: slots.SignalEpisodeMetrics{CompleteCount: 2, HitCount: 1, IncompleteCount: 0},
	}
	report := renderSignalDiagnoseReport(flow)
	for _, want := range []string{
		"**总计波段**：3 段",
		"**盈利波段**：2 段",
		"+10.00%", // 8%+2% cumulative gain on buy
		"**亏损波段**：1 段",
		"1.00%", // buy loss magnitude
		"**总计波段**：2 段",
		"**盈利波段**：1 段",
		"+4.00%",
		"6.00%", // sell loss magnitude
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func TestRenderSignalDiagnoseReportIgnoresLLMComposeSectionsInChat(t *testing.T) {
	flow := flowWithFullEval()
	flow.EvalJudgment = "### 一、信号触发情况\n\n触发密集。\n\n### 二、probe 与命中率\n\n买15卖16。"
	flow.KBDraft = assembleSignalDiagnoseDoc(flow, flow.EvalJudgment)
	report := renderSignalDiagnoseReport(flow)
	if strings.Contains(report, "触发密集") || strings.Contains(report, "买15卖16") {
		t.Fatalf("chat body must not surface KB compose sections:\n%s", report)
	}
	if !strings.Contains(report, "#### 买入（做多）") {
		t.Fatalf("expected wave cards:\n%s", report)
	}
}

func TestAssembleSignalDiagnoseDocKeepsFullDetail(t *testing.T) {
	flow := flowWithFullEval()
	doc := assembleSignalDiagnoseDoc(flow, "Agent 综合判断正文。")
	for _, want := range []string{
		"命中率与有效性怎么算",
		"汇总 · Strict",
		"Agent 评价",
		"Agent 综合判断正文。",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("KB doc missing %q:\n%s", want, doc)
		}
	}
}

func flowWithFullEval() *Flow {
	return &Flow{
		StrategyQuery: "SAR",
		CatalogType:   "index",
		StockName:     "腾讯控股",
		StockCode:     "00700.HK",
		MonthsBack:    3,
		ProbeBuyHits:  15,
		ProbeSellHits: 16,
		Strategies:    []StrategyItem{{Frequency: "60m"}},
		SignalEval: slots.SignalEpisodeEval{
			Method: "until_opposite_signal",
			BuyDetails: []slots.SignalEpisodeDetail{
				{Complete: true, Hit: true, DirectionReturn: 0.05},
				{Complete: true, Hit: false, DirectionReturn: -0.02},
			},
			SellDetails: []slots.SignalEpisodeDetail{
				{Complete: true, Hit: true, DirectionReturn: 0.03},
			},
			BuyEpisodes: slots.SignalEpisodeMetrics{
				CompleteCount: 2, HitCount: 1, HitRate: 0.5,
			},
			SellEpisodes: slots.SignalEpisodeMetrics{
				CompleteCount: 1, HitCount: 1, HitRate: 1,
			},
		},
		DiagnoseRaw: map[string]any{
			"buy_rules": []any{map[string]any{
				"index": "SAR", "type": "signal", "trigger_count": 15,
				"sample_reasons": []any{map[string]any{"time": "2026-01-01", "reason": "flip"}},
			}},
		},
	}
}
