package klinepattern

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func mtfEntry(freq string, score float64, withHits bool) slots.KlinePatternMTFEntry {
	b := slots.KlinePatternBias{Score: score, Bias: "中性", BiasColor: "gray"}
	if score > 0 {
		b.Bias = "看多"
		b.BiasColor = "green"
	} else if score < 0 {
		b.Bias = "看空"
		b.BiasColor = "red"
	}
	e := slots.KlinePatternMTFEntry{
		Frequency: freq,
		Label:     timeframeDisplayLabel(freq),
		Bias:      &b,
	}
	if withHits {
		e.ThreeBar = &slots.KlinePatternThreeBarWindow{
			Bars: []slots.KlinePatternBarWindowEntry{
				{KlinePatterns: []slots.KlinePatternHit{{Name: "Hammer", Direction: 1}}},
			},
		}
	}
	return e
}

func TestComputeKlinePatternMTFWeightedConclusion_ScoreAndBias(t *testing.T) {
	// 0.4*1 + 0.3*(-1) + 0.2*0 + 0.1*1 = 0.2
	entries := []slots.KlinePatternMTFEntry{
		mtfEntry("5m", 1, true),
		mtfEntry("15m", -1, true),
		mtfEntry("30m", 0, false),
		mtfEntry("60m", 1, true),
	}
	bias, summary, ok := ComputeKlinePatternMTFWeightedConclusion(entries)
	if !ok || summary == "" {
		t.Fatalf("ok=%v summary=%q", ok, summary)
	}
	if bias.Score != 0.2 || bias.Bias != "看多" || bias.BiasColor != "green" {
		t.Fatalf("bias=%+v", bias)
	}
}

func TestComputeKlinePatternMTFWeightedConclusion_TieZero(t *testing.T) {
	entries := []slots.KlinePatternMTFEntry{
		mtfEntry("5m", 1, true),
		mtfEntry("15m", -1, true),
		mtfEntry("30m", 0, false),
		mtfEntry("60m", 0, false),
	}
	// 0.4 - 0.3 = 0.1
	bias, _, ok := ComputeKlinePatternMTFWeightedConclusion(entries)
	if !ok || bias.Score != 0.1 {
		t.Fatalf("ok=%v bias=%+v", ok, bias)
	}

	entries2 := []slots.KlinePatternMTFEntry{
		mtfEntry("5m", 1, true),
		mtfEntry("15m", -1, true),
		mtfEntry("30m", -1, true),
		mtfEntry("60m", 2, true), // only +1 counts per hit scoring; use -1 for zero
	}
	entries2[3].Bias = &slots.KlinePatternBias{Score: -1, Bias: "看空", BiasColor: "red"}
	// 0.4 -0.3 -0.2 -0.1 = -0.2
	bias2, _, ok2 := ComputeKlinePatternMTFWeightedConclusion(entries2)
	if !ok2 || bias2.Score != -0.2 || bias2.Bias != "看空" {
		t.Fatalf("bias2=%+v ok=%v", bias2, ok2)
	}
}

func TestComputeKlinePatternMTFWeightedConclusion_SingleTFNonZero(t *testing.T) {
	entries := []slots.KlinePatternMTFEntry{
		mtfEntry("5m", 0, false),
		mtfEntry("15m", 0, false),
		mtfEntry("30m", -1, true),
		mtfEntry("60m", 0, false),
	}
	bias, summary, ok := ComputeKlinePatternMTFWeightedConclusion(entries)
	if !ok || summary == "" {
		t.Fatalf("ok=%v summary=%q", ok, summary)
	}
	if bias.Score != -0.2 || bias.Bias != "看空" {
		t.Fatalf("bias=%+v", bias)
	}
}

func TestComputeKlinePatternMTFWeightedConclusion_NoHits(t *testing.T) {
	entries := []slots.KlinePatternMTFEntry{
		mtfEntry("5m", 0, false),
		mtfEntry("15m", 0, false),
		mtfEntry("30m", 0, false),
		mtfEntry("60m", 0, false),
	}
	_, _, ok := ComputeKlinePatternMTFWeightedConclusion(entries)
	if ok {
		t.Fatal("expected ok=false when no hits and zero score")
	}
}

