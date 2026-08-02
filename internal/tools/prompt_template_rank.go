package tools

import (
	"sort"
	"strings"
)

// techPromptFocus controls how type=tech templates are ordered for the model.
type techPromptFocus string

const (
	techFocusPriceTrend  techPromptFocus = "price_trend"
	techFocusCapitalFlow techPromptFocus = "capital_flow"
)

func parseTechPromptFocus(raw string) techPromptFocus {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "capital_flow", "capital", "flow", "资金":
		return techFocusCapitalFlow
	default:
		return techFocusPriceTrend
	}
}

func promptItemTags(item map[string]any) []string {
	raw, ok := item["tag"].([]any)
	if !ok {
		if ss, ok := item["tag"].([]string); ok {
			out := make([]string, 0, len(ss))
			for _, s := range ss {
				if t := strings.ToLower(strings.TrimSpace(s)); t != "" {
					out = append(out, t)
				}
			}
			return out
		}
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.ToLower(strings.TrimSpace(s)))
		}
	}
	return out
}

func promptItemNameCN(item map[string]any) string {
	if s, ok := item["name_cn"].(string); ok && strings.TrimSpace(s) != "" {
		return strings.TrimSpace(s)
	}
	name, _ := item["name"].(map[string]any)
	if name == nil {
		return ""
	}
	for _, key := range []string{"cn", "zh", "zh_cn"} {
		if s, ok := name[key].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func promptItemID(item map[string]any) string {
	for _, key := range []string{"prompt_id", "promptId", "_id"} {
		if s, ok := item[key].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func techPromptRank(tags []string, focus techPromptFocus) int {
	if len(tags) == 0 {
		if focus == techFocusCapitalFlow {
			return 50
		}
		return 40
	}
	best := 90
	for _, tag := range tags {
		p := techTagPriority(tag, focus)
		if p < best {
			best = p
		}
	}
	return best
}

func techTagPriority(tag string, focus techPromptFocus) int {
	tag = strings.ToLower(strings.TrimSpace(tag))
	if focus == techFocusCapitalFlow {
		switch tag {
		case "capital_flow", "capital", "flow":
			return 1
		case "flag":
			return 20
		case "kline":
			return 21
		case "price":
			return 22
		default:
			return 40
		}
	}
	switch tag {
	case "flag":
		return 1
	case "kline":
		return 2
	case "price":
		return 3
	case "template":
		return 10
	case "industry", "competitor", "etf":
		return 30
	case "capital_flow", "capital", "flow":
		return 99
	case "index":
		return 95
	default:
		return 50
	}
}

type rankedPromptItem struct {
	item map[string]any
	rank int
	idx  int
}

func rankTechPromptItems(items []any, focus techPromptFocus) []any {
	if len(items) == 0 {
		return items
	}
	ranked := make([]rankedPromptItem, 0, len(items))
	for i, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ranked = append(ranked, rankedPromptItem{
			item: item,
			rank: techPromptRank(promptItemTags(item), focus),
			idx:  i,
		})
	}
	if len(ranked) == 0 {
		return items
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].rank != ranked[j].rank {
			return ranked[i].rank < ranked[j].rank
		}
		return ranked[i].idx < ranked[j].idx
	})
	out := make([]any, 0, len(ranked))
	for _, r := range ranked {
		out = append(out, r.item)
	}
	return out
}

func applyTechPromptRouting(data map[string]any, focus techPromptFocus) (map[string]any, string) {
	return processPromptTemplateResponse(data, focus, true, "")
}
