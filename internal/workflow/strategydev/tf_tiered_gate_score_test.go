package strategydev

import (
	"math"
	"strings"
	"testing"
)

func TestTFTieredGate60mWeightsExact(t *testing.T) {
	pa := PriceAnalysis{
		Rows: []PriceAnalysisRow{
			rowWeakDown("周线"),
			rowWeakUp("日线"),
			rowSubscoreMinusOnePointFive("60m"),
		},
	}
	tiered := ComputeTFTieredGatePriceScore(pa, "60m")
	// 0.55*(-0.5) + 0.45*(0.5) = -0.05 → 100 * (-0.05/2) = -2.5
	if tiered.Score != -2.5 {
		t.Fatalf("60m tiered mixed = %v want -2.5", tiered.Score)
	}
	if _, ok := tiered.TFScores["60m"]; ok {
		t.Fatal("60m tiered gate must not include entry 60m subscore")
	}
	allWeak := PriceAnalysis{
		Rows: []PriceAnalysisRow{rowWeakDown("周线"), rowWeakDown("日线"), rowWeakDown("60m")},
	}
	tiered = ComputeTFTieredGatePriceScore(allWeak, "60m")
	if tiered.Score != -25 {
		t.Fatalf("60m tiered all weak = %v want -25", tiered.Score)
	}
}

func TestTFTieredGate5mWeightsExact(t *testing.T) {
	pa := PriceAnalysis{
		Rows: []PriceAnalysisRow{
			rowWeakDown("周线"),
			rowWeakUp("日线"),
			rowSubscoreMinusOnePointFive("60m"),
		},
	}
	tiered := ComputeTFTieredGatePriceScore(pa, "5m")
	// 0.45*(-0.5) + 0.35*(0.5) + 0.20*(-1.5) = -0.225 + 0.175 - 0.30 = -0.35
	want := NormalizePriceScore(100 * (-0.35 / 2))
	if tiered.Score != want {
		t.Fatalf("5m tiered mixed = %v want %v", tiered.Score, want)
	}
}

func TestTFTieredNoRenormalizeWhenTFMissing(t *testing.T) {
	pa := PriceAnalysis{
		Rows: []PriceAnalysisRow{
			{Timeframe: "日线", Conclusion: RegimeUp, Position: "中部", BosChoch: "无"},
		},
	}
	tiered60 := ComputeTFTieredGatePriceScore(pa, "60m")
	// weekly missing: only daily subscore 1.0 * 0.45 = 0.45 → score 22.5 (not 50 from renormalize)
	if tiered60.Score != 22.5 {
		t.Fatalf("60m missing weekly = %v want 22.5 (no renormalize)", tiered60.Score)
	}
	if !strings.Contains(tiered60.ScoreReason, "周线") || !strings.Contains(tiered60.ScoreReason, "缺") {
		t.Fatalf("score_reason should mention missing 周线: %q", tiered60.ScoreReason)
	}

	pa5 := PriceAnalysis{
		Rows: []PriceAnalysisRow{
			{Timeframe: "周线", Conclusion: RegimeUp, Position: "中部", BosChoch: "无"},
			{Timeframe: "日线", Conclusion: RegimeUp, Position: "中部", BosChoch: "无"},
		},
	}
	tiered5 := ComputeTFTieredGatePriceScore(pa5, "5m")
	// 0.45*1 + 0.35*1 = 0.80 → 40
	if tiered5.Score != 40 {
		t.Fatalf("5m missing 60m = %v want 40", tiered5.Score)
	}
}

func TestTFTieredDailyFallsBackToHTFOnly(t *testing.T) {
	pa := PriceAnalysis{
		Rows: []PriceAnalysisRow{
			{Timeframe: "周线", Conclusion: RegimeUp, Position: "中部", BosChoch: "无"},
			{Timeframe: "日线", Conclusion: RegimeDown, Position: "中部", BosChoch: "无"},
		},
	}
	tiered := ComputeTFTieredGatePriceScore(pa, "daily")
	htf := ComputeHTFGatePriceScore(pa, "daily")
	if tiered.Score != htf.Score {
		t.Fatalf("daily tiered=%v htf=%v want same fallback", tiered.Score, htf.Score)
	}
	if tiered.Score != 50 {
		t.Fatalf("daily fallback score = %v want 50", tiered.Score)
	}
}

func TestTFTieredWeeklyFallsBackToHTFOnly(t *testing.T) {
	pa := PriceAnalysis{
		Rows: []PriceAnalysisRow{
			{Timeframe: "周线", Conclusion: RegimeWeakUp, Position: "中部", BosChoch: "无"},
		},
	}
	tiered := ComputeTFTieredGatePriceScore(pa, "weekly")
	htf := ComputeHTFGatePriceScore(pa, "weekly")
	if tiered.Score != htf.Score {
		t.Fatalf("weekly tiered=%v htf=%v", tiered.Score, htf.Score)
	}
}

func TestParsePriceAnalysisGateScoreModeTFTieredAndDefaultUnchanged(t *testing.T) {
	if ParsePriceAnalysisGateScoreMode("") != PriceAnalysisGateScoreModeFull {
		t.Fatal("empty must remain full (default off for tf_tiered)")
	}
	if ParsePriceAnalysisGateScoreMode("tf_tiered") != PriceAnalysisGateScoreModeTFTiered {
		t.Fatal("tf_tiered parse")
	}
	if EffectiveGateScoreMode(PriceAnalysisGateScoreModeTFTiered) != PriceAnalysisGateScoreModeTFTiered {
		t.Fatal("effective tf_tiered")
	}
	if EffectiveGateScoreMode("") != PriceAnalysisGateScoreModeFull {
		t.Fatal("effective empty → full")
	}
}

func TestTFTieredGateIndependentOfEMAStacks(t *testing.T) {
	pa := PriceAnalysis{
		Rows: []PriceAnalysisRow{rowWeakDown("周线"), rowWeakDown("日线")},
	}
	before := ComputeTFTieredGatePriceScore(pa, "60m").Score
	// EMA stacks are display-only; gate score must depend only on PA rows.
	_ = ComputeEMAStacksAtAnchor(nil, nil, nil, "60m", anchorInstantFromDailyBar(map[string]any{"time": float64(1)}))
	after := ComputeTFTieredGatePriceScore(pa, "60m").Score
	if math.Abs(before-after) > 1e-9 {
		t.Fatalf("tiered gate changed without PA change: before=%v after=%v", before, after)
	}
}

func TestTFTieredDiffersFromHTFRenormalizedWhenWeeklyMissing(t *testing.T) {
	pa := PriceAnalysis{
		Rows: []PriceAnalysisRow{
			{Timeframe: "日线", Conclusion: RegimeUp, Position: "中部", BosChoch: "无"},
		},
	}
	tiered := ComputeTFTieredGatePriceScore(pa, "60m")
	htf := ComputeHTFGatePriceScore(pa, "60m")
	if tiered.Score == htf.Score {
		t.Fatalf("tiered=%v should differ from renormalized htf=%v when weekly missing", tiered.Score, htf.Score)
	}
}
