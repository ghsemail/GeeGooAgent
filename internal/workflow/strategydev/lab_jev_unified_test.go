package strategydev

import (
	"math"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestApplyLabJEVUnifiedMirrorsPAJEV(t *testing.T) {
	detail := LabEpisodeDetail{
		PriceAnalysisJEV: &PriceAnalysisJEV{
			OK: true, RecommendedChoice: "看多", RecommendedIndex: 0, RecommendedReason: "结构+形态 JEV 倾向看多（置信 70%）",
			Choices: []PriceAnalysisJEVChoice{{Label: "看多", Probability: 0.7}, {Label: "看空", Probability: 0.2}, {Label: "不明确", Probability: 0.1}},
		},
		KlinePatternSummary: "锤子(看多)",
		KlinePatternBias:    &slots.KlinePatternBias{Score: 1, Bias: "看多", BiasColor: "green"},
	}
	ApplyLabJEVUnified(&detail)
	if detail.JEVUnified == nil {
		t.Fatal("nil jev_unified")
	}
	if detail.JEVUnified.Direction != LabJEVUnifiedBull {
		t.Fatalf("jev_unified=%+v", detail.JEVUnified)
	}
	if math.Abs(detail.JEVUnified.Confidence-0.7) > 1e-9 {
		t.Fatalf("confidence=%v", detail.JEVUnified.Confidence)
	}
	if detail.JEVUnified.Conflict {
		t.Fatal("aligned pattern should not show conflict")
	}
	if detail.JEVUnified.Breakdown.Pattern != "锤子(看多)" {
		t.Fatalf("breakdown=%+v", detail.JEVUnified.Breakdown)
	}
}

func TestApplyLabJEVUnifiedSkipsWhenJEVFailed(t *testing.T) {
	detail := LabEpisodeDetail{
		PriceAnalysisJEV: &PriceAnalysisJEV{OK: false, Error: "timeout"},
		KlinePatternSummary: "x",
	}
	ApplyLabJEVUnified(&detail)
	if detail.JEVUnified != nil {
		t.Fatalf("expected no mirror on failed JEV, got %+v", detail.JEVUnified)
	}
}

func TestLabJEVDisplayConflict(t *testing.T) {
	out := LabEpisodeDetail{
		KlinePatternBias: &slots.KlinePatternBias{Bias: "看多"},
	}
	if !labJEVDisplayConflict(&out, "看空") {
		t.Fatal("expected display conflict")
	}
	if labJEVDisplayConflict(&out, "看多") {
		t.Fatal("same direction should not conflict")
	}
}
