package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCompactPromptTemplateItem_StripsTemplateKeepsBrief(t *testing.T) {
	t.Parallel()
	item := map[string]any{
		"prompt_id": "abc",
		"name":      map[string]any{"cn": "K线形态分析"},
		"intro":     map[string]any{"cn": "我将分析K线形态与支撑阻力。"},
		"tag":       []any{"kline"},
		"creator":   "admin",
		"template":  []any{map[string]any{"Role": "very long prompt body"}},
	}
	compact := compactPromptTemplateItem(item)
	if _, ok := compact["template"]; ok {
		t.Fatal("template should be stripped")
	}
	if compact["brief"] != "我将分析K线形态与支撑阻力。" {
		t.Fatalf("brief=%v", compact["brief"])
	}
	if compact["name_cn"] != "K线形态分析" {
		t.Fatalf("name_cn=%v", compact["name_cn"])
	}
}

func TestProcessPromptTemplateResponse_AdminPreferredOverUserETF(t *testing.T) {
	t.Parallel()
	data := map[string]any{
		"items": []any{
			map[string]any{
				"prompt_id": "etf-user",
				"creator":   "user",
				"name":      map[string]any{"cn": "黄金ETF趋势"},
				"intro":     map[string]any{"cn": "ETF趋势"},
				"tag":       []any{"price", "flag"},
				"template":  []any{map[string]any{"Role": "x"}},
			},
			map[string]any{
				"prompt_id": "kline-admin",
				"creator":   "admin",
				"name":      map[string]any{"cn": "K线形态分析"},
				"intro":     map[string]any{"cn": "K线简介"},
				"tag":       []any{"kline"},
				"template":  []any{map[string]any{"Role": "y"}},
			},
		},
	}
	out, _ := processPromptTemplateResponse(data, techFocusPriceTrend, true, "")
	rec, _ := out["recommended_for_price_trend"].(map[string]any)
	if rec == nil {
		t.Fatal("missing recommendation")
	}
	if rec["prompt_id"] != "kline-admin" {
		t.Fatalf("prompt_id=%v want kline-admin", rec["prompt_id"])
	}
	items, _ := out["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items len=%d", len(items))
	}
	first, _ := items[0].(map[string]any)
	if first["prompt_id"] != "kline-admin" {
		t.Fatalf("first item=%v", first["prompt_id"])
	}
}

func TestProcessPromptTemplateResponse_FitsAgentToolBudget(t *testing.T) {
	t.Parallel()
	items := make([]any, 0, 9)
	for i := 0; i < 9; i++ {
		items = append(items, map[string]any{
			"prompt_id": "id-" + string(rune('a'+i)),
			"creator":   "admin",
			"name":      map[string]any{"cn": "模板名称"},
			"intro":     map[string]any{"cn": "简短简介用于选型"},
			"tag":       []any{"flag"},
			"template":  []any{map[string]any{"Role": strings.Repeat("X", 2000)}},
		})
	}
	out, _ := processPromptTemplateResponse(map[string]any{"items": items}, techFocusPriceTrend, true, "")
	payload := map[string]any{"status": StatusOK, "summary": "x", "data": out}
	raw, _ := json.Marshal(payload)
	if len(raw) > 6000 {
		t.Fatalf("payload still too large: %d bytes", len(raw))
	}
	if !strings.Contains(string(raw), "recommended_for_price_trend") {
		t.Fatal("missing recommendation in payload")
	}
	if strings.Contains(string(raw), `"template"`) {
		t.Fatal("template body should not appear in payload")
	}
}

func TestProcessPromptTemplateResponse_TagPriceSelectsOne(t *testing.T) {
	t.Parallel()
	data := map[string]any{
		"items": []any{
			map[string]any{
				"prompt_id": "price-admin", "creator": "admin",
				"name": map[string]any{"cn": "价格结构分析"}, "tag": []any{"price"},
				"intro": map[string]any{"cn": "价格简介"}, "template": []any{map[string]any{"Role": "x"}},
			},
			map[string]any{
				"prompt_id": "flag-admin", "creator": "admin",
				"name": map[string]any{"cn": "趋势分析"}, "tag": []any{"flag"},
				"intro": map[string]any{"cn": "趋势简介"}, "template": []any{map[string]any{"Role": "y"}},
			},
		},
	}
	out, note := processPromptTemplateResponse(data, techFocusPriceTrend, true, "price")
	if out["selected_prompt_id"] != "price-admin" {
		t.Fatalf("selected=%v", out["selected_prompt_id"])
	}
	items, _ := out["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items len=%d want 1", len(items))
	}
	if !strings.Contains(note, "price-admin") {
		t.Fatalf("note=%q", note)
	}
}

func TestMarshalPromptTemplateResponse_RecommendedBeforeItems(t *testing.T) {
	t.Parallel()
	resp := promptTemplateListResponse{
		RecommendedForPriceTrend: map[string]any{"prompt_id": "p1"},
		Count:                    1,
		Items:                    []map[string]any{{"prompt_id": "p1"}},
	}
	raw, _ := json.Marshal(resp)
	text := string(raw)
	itemsIdx := strings.Index(text, `"items"`)
	recIdx := strings.Index(text, `"recommended_for_price_trend"`)
	if recIdx < 0 || itemsIdx < 0 || recIdx > itemsIdx {
		t.Fatalf("field order wrong: %s", text)
	}
}
