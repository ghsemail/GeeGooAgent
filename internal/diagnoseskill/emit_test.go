package diagnoseskill

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func TestEmitLaneJEVFailureNoDirection(t *testing.T) {
	out := emitLaneJEV(strategydev.PriceAnalysisJEV{OK: false, Error: "jev down"})
	if out.Direction != "" {
		t.Fatalf("direction=%q", out.Direction)
	}
	if out.OK {
		t.Fatal("expected ok=false")
	}
}

func TestDirectionToLedgerNeutral(t *testing.T) {
	if DirectionToLedger("unclear") != "neutral" {
		t.Fatal("unclear should map to neutral")
	}
	if DirectionToLedger("bull") != "long" || DirectionToLedger("bear") != "short" {
		t.Fatal("bull/bear mapping")
	}
}

func TestResultJSONNoRedundantFields(t *testing.T) {
	res := Result{
		"code": "00700.HK", "signal_frequency": "60m",
		"price_analysis": []any{},
		"price_analysis_jev": emitLaneJEV(strategydev.PriceAnalysisJEV{
			OK: true, RecommendedChoice: "看空",
			Choices: []strategydev.PriceAnalysisJEVChoice{{Label: "看空", Probability: 0.7}},
			RecommendedReason: "结构偏空",
		}),
		"jev_unified": emitUnifiedJEV(&strategydev.LabJEVUnified{
			OK: true, Direction: strategydev.LabJEVUnifiedBear,
			RecommendedChoice: "看空", Confidence: 0.74, Conflict: true,
			Choices: []strategydev.PriceAnalysisJEVChoice{{Label: "看空", Probability: 0.74}},
			RecommendedReason: "冲突 cap",
		}),
	}
	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, bad := range []string{"recommended_choice", "recommended_index", "breakdown", "record", "spot_price"} {
		if containsField(s, bad) {
			t.Fatalf("redundant field %q in %s", bad, s)
		}
	}
	if !strings.Contains(s, "bear") || !strings.Contains(s, "direction") {
		t.Fatalf("missing direction in %s", s)
	}
}

func containsField(body, key string) bool {
	return json.Valid([]byte(body)) && (len(key) > 0 && (stringContains(body, "\""+key+"\"")))
}

func stringContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}