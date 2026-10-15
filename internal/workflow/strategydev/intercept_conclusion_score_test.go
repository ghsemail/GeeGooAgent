package strategydev

import "testing"

func TestRuleDerivedInterceptConclusionScoreSoftKeptWhenAnchorMissing(t *testing.T) {
	pa := buildPriceAnalysisWhenPriorTradingDayMissing(100, "60m", KeyLevelBands{})
	sc := ComputeMultiTFPriceScore(pa, "60m")
	msg := RuleDerivedInterceptConclusionScore("buy", false, false, pa, sc, 0)
	if containsIntercepted(msg) {
		t.Fatalf("must not say 已拦截 when soft kept: %q", msg)
	}
	if !stringsContains(msg, "保留") && !stringsContains(msg, "软门控") {
		t.Fatalf("expected soft-keep wording: %q", msg)
	}
}

func containsIntercepted(s string) bool {
	return stringsContains(s, "已拦截")
}

func stringsContains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && stringIndexRunes(s, sub))
}

func stringIndexRunes(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
