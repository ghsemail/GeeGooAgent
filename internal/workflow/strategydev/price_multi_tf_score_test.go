package strategydev

import (
	"strings"
	"testing"
	"time"
)

func TestComputeMultiTFPriceScoreBullish60mGate(t *testing.T) {
	pa := PriceAnalysis{
		Rows: []PriceAnalysisRow{
			{Timeframe: "60m", Conclusion: RegimeUp, Position: "靠近下沿", BosChoch: "BOS 放量"},
			{Timeframe: "日线", Conclusion: RegimeUp, Position: "中部", BosChoch: "无"},
			{Timeframe: "周线", Conclusion: RegimeUp, Position: "中部", BosChoch: "无"},
		},
	}
	sc := ComputeMultiTFPriceScore(pa, "60m")
	if sc.Score <= 0 {
		t.Fatalf("expected positive score, got %.2f", sc.Score)
	}
	if sc.TFScores["60m"].Subscore <= 0 {
		t.Fatalf("60m subscore=%v", sc.TFScores["60m"])
	}
}

func TestScoreGateDoesNotBlockBuyWhen60mUpButDailyUnclear(t *testing.T) {
	pa := PriceAnalysis{
		Rows: []PriceAnalysisRow{
			{Timeframe: "60m", Conclusion: RegimeUp, Position: "中部", BosChoch: "无"},
			{Timeframe: "日线", Conclusion: RegimeUnclear, Position: "中部", BosChoch: "无"},
			{Timeframe: "周线", Conclusion: RegimeUp, Position: "中部", BosChoch: "无"},
		},
	}
	sc := ComputeMultiTFPriceScore(pa, "60m")
	if sc.Score < 0 {
		t.Fatalf("expected non-negative score, got %.2f", sc.Score)
	}
	if ShouldInterceptTriggerByScore("buy", sc.Score, 0) {
		t.Fatal("must not block buy solely because daily unclear while HTF/60m lean up")
	}
	pf := priceFilterForGateFrequency(pa, "60m", map[string][]any{}, nil, time.Time{}, "60m")
	if pf.AllowLong {
		t.Fatal("legacy composed allow_long should still be false on this fixture")
	}
	if !shouldInterceptTrigger("buy", pf) {
		t.Fatal("legacy gate would have blocked buy — score gate replaces this")
	}
}

func TestComputeMultiTFPriceScoreMissingTF(t *testing.T) {
	pa := PriceAnalysis{
		Rows: []PriceAnalysisRow{
			{Timeframe: "60m", Conclusion: RegimeUnclear, Structure: structurePriorTradingDayUnavailable},
		},
	}
	sc := ComputeMultiTFPriceScore(pa, "60m")
	if sc.Score != 0 {
		t.Fatalf("score=%v want 0", sc.Score)
	}
	if !strings.Contains(sc.ScoreReason, "缺周期") {
		t.Fatalf("reason=%q", sc.ScoreReason)
	}
}

func TestShouldInterceptSellOnPositiveScore(t *testing.T) {
	if !ShouldInterceptTriggerByScore("sell", 25, 0) {
		t.Fatal("sell should intercept when score > -threshold and score bullish")
	}
	if ShouldInterceptTriggerByScore("sell", -25, 0) {
		t.Fatal("bearish score should allow sell")
	}
}
