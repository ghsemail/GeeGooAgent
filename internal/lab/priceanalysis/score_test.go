package priceanalysis

import (
	"strings"
	"testing"
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

func TestNormalizePriceScoreNearZeroNotWouldIntercept(t *testing.T) {
	raw := -0.001
	if got := NormalizePriceScore(raw); got != 0 {
		t.Fatalf("NormalizePriceScore(%v)=%v want 0", raw, got)
	}
	if ShouldInterceptTriggerByScore("buy", raw, 0) {
		t.Fatal("buy must not intercept when score rounds to 0")
	}
	reason := formatScoreReason(raw, nil)
	if strings.Contains(reason, "-0") {
		t.Fatalf("score reason must not show -0: %q", reason)
	}
	if !strings.Contains(reason, "0.00") && !strings.Contains(reason, "综合分 0") {
		t.Fatalf("expected zero display in reason: %q", reason)
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
