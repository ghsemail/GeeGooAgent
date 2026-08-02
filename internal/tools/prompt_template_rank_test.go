package tools

import (
	"strings"
	"testing"
)

func TestRankTechPromptItems_PriceTrendPutsCapitalFlowLast(t *testing.T) {
	t.Parallel()
	items := []any{
		map[string]any{"prompt_id": "cf", "name": map[string]any{"cn": "资金流向"}, "tag": []any{"capital_flow"}},
		map[string]any{"prompt_id": "flag", "name": map[string]any{"cn": "趋势"}, "tag": []any{"flag"}},
		map[string]any{"prompt_id": "price", "name": map[string]any{"cn": "价格"}, "tag": []any{"price"}},
	}
	ranked := rankTechPromptItems(items, techFocusPriceTrend)
	if len(ranked) != 3 {
		t.Fatalf("len=%d", len(ranked))
	}
	first := ranked[0].(map[string]any)["prompt_id"]
	last := ranked[2].(map[string]any)["prompt_id"]
	if first != "flag" {
		t.Fatalf("first=%v want flag", first)
	}
	if last != "cf" {
		t.Fatalf("last=%v want cf", last)
	}
}

func TestApplyTechPromptRouting_RecommendedField(t *testing.T) {
	t.Parallel()
	data := map[string]any{
		"items": []any{
			map[string]any{"prompt_id": "cf", "name": map[string]any{"cn": "资金流向"}, "tag": []any{"capital_flow"}},
			map[string]any{"prompt_id": "kline", "name": map[string]any{"cn": "K线形态"}, "tag": []any{"kline"}},
		},
	}
	out, note := applyTechPromptRouting(data, techFocusPriceTrend)
	rec, _ := out["recommended_for_price_trend"].(map[string]any)
	if rec == nil {
		t.Fatal("missing recommended_for_price_trend")
	}
	if rec["prompt_id"] != "kline" {
		t.Fatalf("prompt_id=%v", rec["prompt_id"])
	}
	if !strings.Contains(note, "kline") && !strings.Contains(note, "K线形态") {
		t.Fatalf("note=%q", note)
	}
	items, _ := out["items"].([]any)
	first, _ := items[0].(map[string]any)
	if _, ok := first["template"]; ok {
		t.Fatal("items should be compact")
	}
}
