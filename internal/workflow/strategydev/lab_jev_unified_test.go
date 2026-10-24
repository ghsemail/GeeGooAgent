package strategydev

import (
	"math"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestSynthesizeLabJEVUnifiedAlignedBull(t *testing.T) {
	structure := jevUnifiedLeg{dir: LabJEVUnifiedBull, confidence: 0.8, summary: "结构看多", present: true}
	pattern := jevUnifiedLeg{dir: LabJEVUnifiedBull, confidence: 0.5, summary: "锤子(看多)", present: true}
	got := synthesizeLabJEVUnified(structure, pattern)
	if got.Direction != LabJEVUnifiedBull || got.Conflict {
		t.Fatalf("got %+v", got)
	}
	wantScore := 0.7*0.8 + 0.3*0.5
	if math.Abs(got.Confidence-wantScore) > 1e-9 {
		t.Fatalf("confidence=%v want |score|=%v", got.Confidence, wantScore)
	}
}

func TestSynthesizeLabJEVUnifiedOppositeStillBearWithConflict(t *testing.T) {
	structure := jevUnifiedLeg{dir: LabJEVUnifiedBear, confidence: 0.9, summary: "结构看空", present: true}
	pattern := jevUnifiedLeg{dir: LabJEVUnifiedBull, confidence: 0.3, summary: "弱反弹(看多)", present: true}
	got := synthesizeLabJEVUnified(structure, pattern)
	if got.Direction != LabJEVUnifiedBear || !got.Conflict {
		t.Fatalf("expected bear with conflict, got %+v", got)
	}
	want := math.Abs(-0.7*0.9 + 0.3*0.3)
	if math.Abs(got.Confidence-want) > 1e-9 {
		t.Fatalf("confidence=%v want %v", got.Confidence, want)
	}
}

func TestSynthesizeLabJEVUnifiedSmallScoreStillDirected(t *testing.T) {
	structure := jevUnifiedLeg{dir: LabJEVUnifiedBear, confidence: 0.4, summary: "结构偏空", present: true}
	pattern := jevUnifiedLeg{dir: LabJEVUnifiedBull, confidence: 0.5, summary: "形态偏多", present: true}
	got := synthesizeLabJEVUnified(structure, pattern)
	score := -0.7*0.4 + 0.3*0.5
	if math.Abs(score-(-0.13)) > 1e-9 {
		t.Fatalf("score=%v", score)
	}
	if got.Direction != LabJEVUnifiedBear || !got.Conflict {
		t.Fatalf("expected weak bear with conflict, got %+v", got)
	}
	if math.Abs(got.Confidence-0.13) > 1e-9 {
		t.Fatalf("confidence=%v want 0.13", got.Confidence)
	}
}

func TestSynthesizeLabJEVUnifiedExactCancelUnclear(t *testing.T) {
	structure := jevUnifiedLeg{dir: LabJEVUnifiedBear, confidence: 0.3, summary: "结构偏空", present: true}
	pattern := jevUnifiedLeg{dir: LabJEVUnifiedBull, confidence: 0.7, summary: "形态偏多", present: true}
	got := synthesizeLabJEVUnified(structure, pattern)
	if got.Direction != LabJEVUnifiedUnclear || !got.Conflict {
		t.Fatalf("got %+v", got)
	}
	if got.Confidence != 0 {
		t.Fatalf("confidence=%v want 0", got.Confidence)
	}
}

func TestSynthesizeLabJEVUnifiedStructureUnclearWithPatternConflict(t *testing.T) {
	structure := jevUnifiedLeg{dir: LabJEVUnifiedUnclear, confidence: 0.6, summary: "结构不明确", present: true}
	pattern := jevUnifiedLeg{dir: LabJEVUnifiedBull, confidence: 0.7, summary: "形态偏多", present: true}
	got := synthesizeLabJEVUnified(structure, pattern)
	if got.Direction != LabJEVUnifiedBull || !got.Conflict {
		t.Fatalf("got %+v", got)
	}
	want := 0.3 * 0.7
	if math.Abs(got.Confidence-want) > 1e-9 {
		t.Fatalf("confidence=%v want %v", got.Confidence, want)
	}
}

func TestSynthesizeLabJEVUnifiedMissingPatternUsesStructureScore(t *testing.T) {
	structure := jevUnifiedLeg{dir: LabJEVUnifiedBear, confidence: 0.75, summary: "结构看空", present: true}
	pattern := jevUnifiedLeg{present: false, summary: "无本地形态数据"}
	got := synthesizeLabJEVUnified(structure, pattern)
	want := 0.7 * 0.75
	if got.Direction != LabJEVUnifiedBear || got.Conflict {
		t.Fatalf("got %+v", got)
	}
	if math.Abs(got.Confidence-want) > 1e-9 {
		t.Fatalf("confidence=%v want %v", got.Confidence, want)
	}
}

func TestSynthesizeLabJEVUnifiedMissingStructure(t *testing.T) {
	structure := jevUnifiedLeg{present: false, summary: "结构 JEV 不可用"}
	pattern := jevUnifiedLeg{dir: LabJEVUnifiedBull, confidence: 0.5, summary: "形态", present: true}
	got := synthesizeLabJEVUnified(structure, pattern)
	if got.Direction != LabJEVUnifiedUnclear || !got.Conflict || got.Confidence != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestApplyLabJEVUnifiedFromDetailFields(t *testing.T) {
	detail := LabEpisodeDetail{
		PriceAnalysisJEV: &PriceAnalysisJEV{
			OK: true, RecommendedChoice: "看多", RecommendedIndex: 0, RecommendedReason: "多周期结构倾向看多（置信 70%）",
			Choices: []PriceAnalysisJEVChoice{{Label: "看多", Probability: 0.7}, {Label: "看空", Probability: 0.2}, {Label: "不明确", Probability: 0.1}},
		},
		KlinePatternSummary: "锤子(看多)",
		KlinePatternBias:    &slots.KlinePatternBias{Score: 1, Bias: "看多", BiasColor: "green"},
		KlinePatterns:       []slots.KlinePatternHit{{Name: "hammer", Direction: 1, LabelCN: "锤子"}},
	}
	ApplyLabJEVUnified(&detail)
	if detail.JEVUnified == nil {
		t.Fatal("nil jev_unified")
	}
	if detail.JEVUnified.Direction != LabJEVUnifiedBull || detail.JEVUnified.Conflict {
		t.Fatalf("jev_unified=%+v", detail.JEVUnified)
	}
	if detail.JEVUnified.Breakdown.Structure == "" || detail.JEVUnified.Breakdown.Pattern == "" {
		t.Fatalf("breakdown=%+v", detail.JEVUnified.Breakdown)
	}
}

func TestPatternConfidenceFromHitsRatio(t *testing.T) {
	hits := []slots.KlinePatternHit{
		{Direction: 1}, {Direction: 1}, {Direction: -1},
	}
	conf := patternConfidenceFromBias(nil, hits)
	if math.Abs(conf-2.0/3.0) > 1e-9 {
		t.Fatalf("conf=%v", conf)
	}
}

func TestDirectionConfidenceFromScoreSignOnly(t *testing.T) {
	dir, conf := directionConfidenceFromScore(0.14)
	if dir != LabJEVUnifiedBull || conf != 0.14 {
		t.Fatalf("got %v %v", dir, conf)
	}
	dir, conf = directionConfidenceFromScore(-0.2)
	if dir != LabJEVUnifiedBear || conf != 0.2 {
		t.Fatalf("got %v %v", dir, conf)
	}
	dir, conf = directionConfidenceFromScore(0)
	if dir != LabJEVUnifiedUnclear || conf != 0 {
		t.Fatalf("got %v %v", dir, conf)
	}
}
