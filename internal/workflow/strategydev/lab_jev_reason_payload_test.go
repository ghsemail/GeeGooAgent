package strategydev

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSummarizePriceAnalysisJEVReasonPayloadIncludesFacts(t *testing.T) {
	wantFacts := "weekly 结构偏多"
	var capturedUser string
	jev := NewJevClient(JevConfig{APIKey: "test-key"})
	jev.SetReasonChat(func(ctx context.Context, system, user string) (string, error) {
		capturedUser = user
		return "多周期结构同向偏多。", nil
	})
	choices := []PriceAnalysisJEVChoice{{Label: "看多", Probability: 0.7}}
	if _, err := jev.SummarizePriceAnalysisJEVReason(context.Background(), "看多", 0, choices, 0.7, wantFacts); err != nil {
		t.Fatal(err)
	}
	var payload priceAnalysisJEVReasonPayload
	if err := json.Unmarshal([]byte(capturedUser), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Facts != wantFacts {
		t.Fatalf("facts=%q want=%q", payload.Facts, wantFacts)
	}
	if payload.RecommendedChoice != "看多" {
		t.Fatalf("choice=%q", payload.RecommendedChoice)
	}
}

func TestSummarizeKlineJEVReasonPayloadIncludesFacts(t *testing.T) {
	wantFacts := "锚点锤子线偏多"
	var capturedUser string
	jev := NewJevClient(JevConfig{APIKey: "test-key"})
	jev.SetReasonChat(func(ctx context.Context, system, user string) (string, error) {
		capturedUser = user
		return "锚点形态偏多。", nil
	})
	if _, err := jev.SummarizeKlineJEVReason(context.Background(), "看多", 0, nil, 0.6, wantFacts); err != nil {
		t.Fatal(err)
	}
	var payload priceAnalysisJEVReasonPayload
	if err := json.Unmarshal([]byte(capturedUser), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Facts != wantFacts {
		t.Fatalf("facts=%q want=%q", payload.Facts, wantFacts)
	}
}

func TestSummarizeTechnicalAnalysisJEVReasonPayloadIncludesFacts(t *testing.T) {
	ta := sampleLabTA00700()
	wantFacts := BuildTechnicalAnalysisJEVState(ta, "sell", "60m")
	if wantFacts == "" {
		t.Fatal("empty ta facts fixture")
	}
	var capturedUser string
	jev := NewJevClient(JevConfig{APIKey: "test-key"})
	jev.SetReasonChat(func(ctx context.Context, system, user string) (string, error) {
		capturedUser = user
		return "日线价值区下方，过滤层中性。", nil
	})
	if _, err := jev.SummarizeTechnicalAnalysisJEVReason(context.Background(), "看空", 1, nil, 0.85, wantFacts); err != nil {
		t.Fatal(err)
	}
	var payload priceAnalysisJEVReasonPayload
	if err := json.Unmarshal([]byte(capturedUser), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Facts != wantFacts {
		t.Fatalf("facts missing or mismatch len=%d", len(payload.Facts))
	}
}

func TestLabJEVUnifiedLegReasonInputsIncludesFacts(t *testing.T) {
	plan := labJEVUnifiedPlan{attemptPA: true, attemptKline: true, attemptTA: true}
	legs := []labJEVLegSnapshot{
		{name: labJEVLegNamePA, facts: "pa-state", choice: "看多", reason: "pa-reason"},
		{name: labJEVLegNameKline, facts: "kline-state", choice: "不明确", reason: "kline-reason"},
		{name: labJEVLegNameTA, facts: "ta-state", choice: "看空", reason: "ta-reason"},
	}
	out := labJEVUnifiedLegReasonInputs(plan, legs)
	if len(out) != 3 {
		t.Fatalf("len=%d", len(out))
	}
	byName := map[string]labJEVUnifiedLegReasonInput{}
	for _, in := range out {
		byName[in.Name] = in
	}
	if byName[labJEVLegNamePA].Facts != "pa-state" || !byName[labJEVLegNamePA].Participated {
		t.Fatalf("pa=%+v", byName[labJEVLegNamePA])
	}
	if byName[labJEVLegNameKline].Facts != "kline-state" {
		t.Fatalf("kline=%+v", byName[labJEVLegNameKline])
	}
	if byName[labJEVLegNameTA].Facts != "ta-state" {
		t.Fatalf("ta=%+v", byName[labJEVLegNameTA])
	}
}

func TestSummarizeLabJEVUnifiedReasonPayloadIncludesLegFacts(t *testing.T) {
	legInputs := []labJEVUnifiedLegReasonInput{
		{Name: labJEVLegNamePA, Participated: true, RecommendedChoice: "看多", RecommendedReason: "结构偏多", Facts: "pa-facts"},
	}
	var capturedUser string
	jev := NewJevClient(JevConfig{APIKey: "test-key"})
	jev.SetReasonChat(func(ctx context.Context, system, user string) (string, error) {
		capturedUser = user
		return "结构与总方向一致偏多。", nil
	})
	if _, err := jev.SummarizeLabJEVUnifiedReason(context.Background(), "看多", 0, nil, 0.7, legInputs); err != nil {
		t.Fatal(err)
	}
	var payload labJEVUnifiedReasonPayload
	if err := json.Unmarshal([]byte(capturedUser), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.LegConclusions) != 1 || payload.LegConclusions[0].Facts != "pa-facts" {
		t.Fatalf("leg_conclusions=%+v", payload.LegConclusions)
	}
}
