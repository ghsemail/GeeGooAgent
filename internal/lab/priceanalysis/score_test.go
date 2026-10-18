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

func TestComputeMultiTFPriceScore5mGateUses60mNotEntry(t *testing.T) {
	pa := PriceAnalysis{
		Rows: []PriceAnalysisRow{
			{Timeframe: "5m", Conclusion: RegimeDown, Position: "靠近上沿", BosChoch: "无"},
			{Timeframe: "60m", Conclusion: RegimeUp, Position: "中部", BosChoch: "无"},
			{Timeframe: "日线", Conclusion: RegimeUp, Position: "中部", BosChoch: "无"},
			{Timeframe: "周线", Conclusion: RegimeUp, Position: "中部", BosChoch: "无"},
		},
	}
	sc := ComputeMultiTFPriceScore(pa, "5m")
	if _, ok := sc.TFScores["5m"]; ok {
		t.Fatal("5m must not appear in tf_scores")
	}
	for _, label := range []string{"60m", "日线", "周线"} {
		if sc.TFScores[label].Missing {
			t.Fatalf("%s missing in tf_scores", label)
		}
	}
	// 60m-only bullish subscore (1.0) with flat HTFs → weight 0.20 of composite.
	pa60Only := PriceAnalysis{
		Rows: []PriceAnalysisRow{
			{Timeframe: "5m", Conclusion: RegimeDown, Position: "靠近上沿", BosChoch: "无"},
			{Timeframe: "60m", Conclusion: RegimeUp, Position: "中部", BosChoch: "无"},
			{Timeframe: "日线", Conclusion: RegimeUnclear, Position: "中部", BosChoch: "无"},
			{Timeframe: "周线", Conclusion: RegimeUnclear, Position: "中部", BosChoch: "无"},
		},
	}
	sc60 := ComputeMultiTFPriceScore(pa60Only, "5m")
	wantScore := NormalizePriceScore(100 * clamp(0.20*1.0/2, -1, 1))
	if sc60.Score != wantScore {
		t.Fatalf("score=%.2f want %.2f (60m weight 0.20)", sc60.Score, wantScore)
	}
	_ = sc
}

func TestComputeMultiTFPriceScoreDailyGateRenormalizesNo60mMissing(t *testing.T) {
	pa := PriceAnalysis{
		Rows: []PriceAnalysisRow{
			{Timeframe: "日线", Conclusion: RegimeUp, Position: "中部", BosChoch: "无"},
			{Timeframe: "周线", Conclusion: RegimeUp, Position: "中部", BosChoch: "无"},
		},
	}
	sc := ComputeMultiTFPriceScore(pa, "daily")
	if _, ok := sc.TFScores["60m"]; ok {
		t.Fatal("daily signal must not include 60m in tf_scores")
	}
	if strings.Contains(sc.ScoreReason, "60m") {
		t.Fatalf("must not report missing 60m: %q", sc.ScoreReason)
	}
	if !strings.Contains(sc.ScoreReason, "综合周期权重") {
		t.Fatalf("expected weight note: %q", sc.ScoreReason)
	}
	if !strings.Contains(sc.ScoreReason, "周线 56.2%") || !strings.Contains(sc.ScoreReason, "日线 43.8%") {
		t.Fatalf("renormalized weights: %q", sc.ScoreReason)
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
