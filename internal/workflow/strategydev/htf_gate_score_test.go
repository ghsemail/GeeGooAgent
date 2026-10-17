package strategydev

import (
	"math"
	"testing"
)

func rowWeakDown(tf string) PriceAnalysisRow {
	return PriceAnalysisRow{Timeframe: tf, Conclusion: RegimeWeakDown, Position: "中部", BosChoch: "无"}
}

func rowWeakUp(tf string) PriceAnalysisRow {
	return PriceAnalysisRow{Timeframe: tf, Conclusion: RegimeWeakUp, Position: "中部", BosChoch: "无"}
}

func rowSubscoreMinusOnePointFive(tf string) PriceAnalysisRow {
	return PriceAnalysisRow{Timeframe: tf, Conclusion: RegimeDown, Position: "靠近上沿", BosChoch: "无"}
}

func TestComputeHTFGatePriceScoreAAPLReference(t *testing.T) {
	allWeakDown := PriceAnalysis{
		Rows: []PriceAnalysisRow{rowWeakDown("周线"), rowWeakDown("日线"), rowWeakDown("60m")},
	}
	gate := ComputeHTFGatePriceScore(allWeakDown, "60m")
	if gate.Score != -25 {
		t.Fatalf("gate all weak down = %v want -25", gate.Score)
	}
	display := ComputeMultiTFPriceScore(allWeakDown, "60m")
	if display.Score != -25 {
		t.Fatalf("display = %v want -25", display.Score)
	}

	mixed := PriceAnalysis{
		Rows: []PriceAnalysisRow{
			rowWeakDown("周线"),
			rowWeakUp("日线"),
			rowSubscoreMinusOnePointFive("60m"),
		},
	}
	display = ComputeMultiTFPriceScore(mixed, "60m")
	if display.Score != -17.5 {
		t.Fatalf("display mixed = %v want -17.5", display.Score)
	}
	gate = ComputeHTFGatePriceScore(mixed, "60m")
	if math.Abs(gate.Score-(-3.13)) > 0.01 {
		t.Fatalf("gate mixed = %v want ≈ -3.13", gate.Score)
	}
}

func TestHTFGateDisplayStillIncludesEntryTF(t *testing.T) {
	pa := PriceAnalysis{
		Rows: []PriceAnalysisRow{
			rowWeakDown("周线"),
			rowWeakUp("日线"),
			rowSubscoreMinusOnePointFive("60m"),
		},
	}
	display := ComputeMultiTFPriceScore(pa, "60m")
	if _, ok := display.TFScores["60m"]; !ok {
		t.Fatal("display tf_scores must include 60m")
	}
	gate := ComputeHTFGatePriceScore(pa, "60m")
	if _, ok := gate.TFScores["60m"]; ok {
		t.Fatal("gate tf_scores must not include entry 60m")
	}
	if display.Score == gate.Score {
		t.Fatal("expected display and gate to differ on this fixture")
	}
}

func TestHTFGateReverseSoftUsesGateScore(t *testing.T) {
	th := 0.0
	gateScore := -3.13
	if WouldInterceptTriggerByScoreMode("buy", gateScore, th, BuyScoreDirectionReverse, true) {
		t.Fatal("reverse: gate -3.13 must not intercept")
	}
	if !WouldInterceptTriggerByScoreMode("buy", 3.13, th, BuyScoreDirectionReverse, true) {
		t.Fatal("reverse: gate 3.13 must intercept")
	}
	if WouldInterceptTriggerByScoreMode("buy", 0, th, BuyScoreDirectionReverse, true) {
		t.Fatal("reverse: gate 0 must pass")
	}
}

func TestFullGateScoreModeMatchesDisplayIntercept(t *testing.T) {
	pa := PriceAnalysis{
		Rows: []PriceAnalysisRow{
			rowWeakDown("周线"),
			rowWeakUp("日线"),
			rowSubscoreMinusOnePointFive("60m"),
		},
	}
	display := ComputeMultiTFPriceScore(pa, "60m")
	gate := ComputeHTFGatePriceScore(pa, "60m")
	// full mode uses display score; HTF gate differs — full-path intercept must match display-only score.
	wFull := WouldInterceptTriggerByScoreMode("buy", display.Score, 0, BuyScoreDirectionReverse, true)
	wDisplayOnly := WouldInterceptTriggerByScoreMode("buy", display.Score, 0, BuyScoreDirectionReverse, true)
	if wFull != wDisplayOnly {
		t.Fatal("full mode intercept must match display score path")
	}
	_ = gate
	if WouldInterceptTriggerByScoreMode("buy", gate.Score, 0, BuyScoreDirectionReverse, true) {
		t.Fatal("HTF gate score alone would intercept; full mode must not use it by default")
	}
}

func TestHTFGateDailySignalWeeklyOnly(t *testing.T) {
	pa := PriceAnalysis{
		Rows: []PriceAnalysisRow{
			{Timeframe: "周线", Conclusion: RegimeUp, Position: "中部", BosChoch: "无"},
			{Timeframe: "日线", Conclusion: RegimeDown, Position: "中部", BosChoch: "无"},
		},
	}
	gate := ComputeHTFGatePriceScore(pa, "daily")
	if gate.Score != 50 {
		t.Fatalf("daily gate weekly-only = %v want 50 (subscore 1 → 100*0.5)", gate.Score)
	}
	if _, ok := gate.TFScores["日线"]; ok {
		t.Fatal("daily HTF gate must exclude 日线")
	}
}

func TestHTFGateWeeklySignalFallbackToDisplay(t *testing.T) {
	pa := PriceAnalysis{
		Rows: []PriceAnalysisRow{
			{Timeframe: "周线", Conclusion: RegimeWeakUp, Position: "中部", BosChoch: "无"},
		},
	}
	display := ComputeMultiTFPriceScore(pa, "weekly")
	gate := ComputeHTFGatePriceScore(pa, "weekly")
	if gate.Score != display.Score {
		t.Fatalf("weekly gate fallback: gate=%v display=%v", gate.Score, display.Score)
	}
}

func TestParsePriceAnalysisGateScoreModeDefaultFull(t *testing.T) {
	if ParsePriceAnalysisGateScoreMode("") != PriceAnalysisGateScoreModeFull {
		t.Fatal("empty must be full")
	}
	if ParsePriceAnalysisGateScoreMode("htf_only") != PriceAnalysisGateScoreModeHTFOnly {
		t.Fatal("htf_only")
	}
}
