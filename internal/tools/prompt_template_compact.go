package tools

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// promptTemplateListResponse keeps recommendation fields before items in JSON output.
type promptTemplateListResponse struct {
	RecommendedForPriceTrend  map[string]any   `json:"recommended_for_price_trend,omitempty"`
	RecommendedForCapitalFlow map[string]any   `json:"recommended_for_capital_flow,omitempty"`
	Count                     int              `json:"count"`
	Items                     []map[string]any `json:"items"`
}

func promptItemCreator(item map[string]any) string {
	c, _ := item["creator"].(string)
	return strings.ToLower(strings.TrimSpace(c))
}

func promptItemIntroCN(item map[string]any) string {
	intro, _ := item["intro"].(map[string]any)
	if intro == nil {
		return ""
	}
	for _, key := range []string{"cn", "zh", "zh_cn"} {
		if s, ok := intro[key].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func promptItemBrief(item map[string]any) string {
	if brief := strings.TrimSpace(strArg(item, "brief", "")); brief != "" {
		return brief
	}
	return promptItemIntroCN(item)
}

func compactPromptTemplateItem(item map[string]any) map[string]any {
	if item == nil {
		return nil
	}
	out := map[string]any{}
	if id := promptItemID(item); id != "" {
		out["prompt_id"] = id
	}
	if name := promptItemNameCN(item); name != "" {
		out["name_cn"] = name
	}
	if brief := promptItemBrief(item); brief != "" {
		out["brief"] = brief
	}
	if tags := promptItemTags(item); len(tags) > 0 {
		out["tag"] = tags
	}
	if creator := promptItemCreator(item); creator != "" {
		out["creator"] = creator
	}
	if t, ok := item["type"].(string); ok && strings.TrimSpace(t) != "" {
		out["type"] = strings.TrimSpace(t)
	}
	if period := item["period"]; period != nil {
		out["period"] = period
	}
	if sw, ok := item["promptSwitch"]; ok {
		out["promptSwitch"] = sw
	} else if sw, ok := item["switch"]; ok {
		out["promptSwitch"] = sw
	}
	return out
}

func promptTemplateItemsFromData(data map[string]any) []any {
	if data == nil {
		return nil
	}
	if items, ok := data["items"].([]any); ok {
		return items
	}
	return nil
}

func compactPromptTemplateItems(items []any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if compact := compactPromptTemplateItem(item); compact != nil {
			out = append(out, compact)
		}
	}
	return out
}

func promptItemCreatorPenalty(item map[string]any) int {
	switch promptItemCreator(item) {
	case "admin":
		return 0
	case "user":
		return 12
	default:
		return 6
	}
}

func techPromptRankItem(item map[string]any, focus techPromptFocus) int {
	return techPromptRank(promptItemTags(item), focus) + promptItemCreatorPenalty(item)
}

func rankCompactPromptItems(items []map[string]any, focus techPromptFocus) []map[string]any {
	if len(items) == 0 {
		return items
	}
	type row struct {
		item map[string]any
		rank int
		idx  int
	}
	rows := make([]row, 0, len(items))
	for i, item := range items {
		rows = append(rows, row{item: item, rank: techPromptRankItem(item, focus), idx: i})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].rank != rows[j].rank {
			return rows[i].rank < rows[j].rank
		}
		return rows[i].idx < rows[j].idx
	})
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.item)
	}
	return out
}

func pickRecommendedPrompt(items []map[string]any, focus techPromptFocus) map[string]any {
	for _, item := range items {
		if techPromptRankItem(item, focus) >= 90 {
			continue
		}
		return item
	}
	if len(items) > 0 {
		return items[0]
	}
	return nil
}

func recommendationRecord(item map[string]any, focus techPromptFocus) map[string]any {
	if item == nil {
		return nil
	}
	rec := map[string]any{
		"prompt_id": promptItemID(item),
		"tags":      promptItemTags(item),
	}
	if name := promptItemNameCN(item); name != "" {
		rec["name_cn"] = name
	}
	if brief := promptItemBrief(item); brief != "" {
		rec["brief"] = brief
	}
	switch focus {
	case techFocusCapitalFlow:
		rec["note"] = "用户明确问资金/主力/净流入时使用"
	default:
		rec["note"] = "用户问价格/走势/涨跌时参考；按 tag 与用户意图在 items 中选取（flag/kline/price），勿默认 capital_flow"
	}
	return rec
}

func processPromptTemplateResponse(data map[string]any, focus techPromptFocus, applyRouting bool) (map[string]any, string) {
	if data == nil {
		return data, ""
	}
	compact := compactPromptTemplateItems(promptTemplateItemsFromData(data))
	if !applyRouting {
		resp := promptTemplateListResponse{Count: len(compact), Items: compact}
		return marshalPromptTemplateResponse(resp), ""
	}

	ranked := rankCompactPromptItems(compact, focus)
	pick := pickRecommendedPrompt(ranked, focus)
	resp := promptTemplateListResponse{Count: len(ranked), Items: ranked}
	var note string
	if pick != nil {
		rec := recommendationRecord(pick, focus)
		id := promptItemID(pick)
		name := promptItemNameCN(pick)
		switch focus {
		case techFocusCapitalFlow:
			resp.RecommendedForCapitalFlow = rec
			if id != "" {
				note = fmt.Sprintf("资金分析推荐 prompt_id=%s（%s）", id, name)
			}
		default:
			resp.RecommendedForPriceTrend = rec
			if id != "" {
				note = fmt.Sprintf("价格/走势参考 prompt_id=%s（%s）", id, name)
			}
		}
	}
	return marshalPromptTemplateResponse(resp), note
}

func marshalPromptTemplateResponse(resp promptTemplateListResponse) map[string]any {
	raw, err := json.Marshal(resp)
	if err != nil {
		return map[string]any{"count": resp.Count, "items": resp.Items}
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{"count": resp.Count, "items": resp.Items}
	}
	return out
}
