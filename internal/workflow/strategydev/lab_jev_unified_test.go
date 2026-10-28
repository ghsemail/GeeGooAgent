package strategydev

import (
	"math"
	"strings"
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
	if !detail.JEVUnified.OK {
		t.Fatalf("ok=false err=%s", detail.JEVUnified.Error)
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
	if detail.JEVUnified.RecommendedChoice != "看多" {
		t.Fatalf("recommended_choice=%q", detail.JEVUnified.RecommendedChoice)
	}
}

func TestApplyLabJEVUnifiedSkipsWhenJEVFailed(t *testing.T) {
	detail := LabEpisodeDetail{
		PriceAnalysisJEV:    &PriceAnalysisJEV{OK: false, Error: "timeout"},
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

func TestBuildLabJEVUnifiedStateFactsAndOmittedLeg(t *testing.T) {
	plan := labJEVUnifiedPlan{attemptPA: true, attemptKline: false, attemptTA: true}
	legs := []labJEVLegSnapshot{{
		name:    labJEVLegNamePA,
		facts:   "标的 TEST\n摘要：偏多",
		choice:  "看多",
		reason:  "结构偏多",
		choices: []PriceAnalysisJEVChoice{{Label: "看多", Probability: 0.8}},
	}, {
		name:    labJEVLegNameTA,
		facts:   "日线 Impulse=绿",
		choice:  "看空",
		reason:  "指标偏空",
		choices: []PriceAnalysisJEVChoice{{Label: "看空", Probability: 0.6}},
	}}
	state := BuildLabJEVUnifiedState(plan, legs, "buy")
	if !strings.Contains(state, "基础事实摘要") {
		t.Fatalf("missing facts: %q", state)
	}
	if !strings.Contains(state, "JEV 结论") {
		t.Fatalf("missing conclusions: %q", state)
	}
	if !strings.Contains(state, "未参与") || !strings.Contains(state, labJEVLegNameKline) {
		t.Fatalf("kline omit missing: %q", state)
	}
	if strings.Contains(state, "K 线形态 JEV") && strings.Count(state, "基础事实摘要") < 2 {
		t.Fatalf("expected facts for PA and TA only: %q", state)
	}
}

func TestLabJEVUnifiedConflictCapsConfidence(t *testing.T) {
	legs := []labJEVLegSnapshot{
		{name: labJEVLegNamePA, choice: "看多"},
		{name: labJEVLegNameKline, choice: "看空"},
	}
	if !labJEVUnifiedLegConflict(legs, "看多") {
		t.Fatal("expected conflict")
	}
	conf := 0.92
	if labJEVUnifiedLegConflict(legs, "看多") && conf > labJEVUnifiedConflictConfidenceCap {
		conf = labJEVUnifiedConflictConfidenceCap
	}
	if conf != 0.75 {
		t.Fatalf("conf=%v", conf)
	}
}
