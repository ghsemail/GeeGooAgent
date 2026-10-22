package chat

import "testing"

func TestDiagnoseBarCountFromRange(t *testing.T) {
	data := map[string]any{
		"range": map[string]any{"bar_count": 490},
	}
	if got := diagnoseBarCount(data); got != 490 {
		t.Fatalf("bar_count=%d", got)
	}
}

func TestDiagnoseDetailPayloadUsableRequiresRulesOrBars(t *testing.T) {
	if DiagnoseDetailPayloadUsable(map[string]any{"verdict": "ok", "summary": "x"}) {
		t.Fatal("empty shell should not be usable")
	}
	if !DiagnoseDetailPayloadUsable(map[string]any{
		"buy_rules": []any{map[string]any{"index": "SAR", "last_bar": map[string]any{"signal": 0}}},
	}) {
		t.Fatal("expected rules usable")
	}
}
