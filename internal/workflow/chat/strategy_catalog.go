package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

const (
	catalogTypeCombination = "combination"
	catalogTypeIndex       = "index"
	catalogTypeCustom      = "custom"
	catalogTypeDefinition  = "definition"
)

type catalogMatch struct {
	Type  string
	Label string
	Raw   map[string]any
}

func resolveStrategyCatalog(
	ctx context.Context,
	query string,
	toolCtx tools.Context,
	runTool ToolRunner,
) (catalogMatch, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return catalogMatch{}, fmt.Errorf("请说明要认知的策略名称")
	}
	try := []func() (catalogMatch, error){
		func() (catalogMatch, error) { return matchCombination(ctx, query, toolCtx, runTool) },
		func() (catalogMatch, error) { return matchIndex(ctx, query, toolCtx, runTool) },
		func() (catalogMatch, error) { return matchCustomSkill(ctx, query, toolCtx, runTool) },
		func() (catalogMatch, error) { return matchDefinition(ctx, query, toolCtx, runTool) },
	}
	var lastErr error
	for _, fn := range try {
		match, err := fn()
		if err == nil {
			return match, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("未在策略库找到「%s」", query)
	}
	return catalogMatch{}, lastErr
}

func matchCombination(ctx context.Context, query string, toolCtx tools.Context, runTool ToolRunner) (catalogMatch, error) {
	res := runTool(ctx, tools.CallRequest{Name: "get_signal_combinations", Arguments: map[string]any{}}, toolCtx)
	if res.Status != tools.StatusOK {
		return catalogMatch{}, fmt.Errorf("get_signal_combinations: %s", res.Summary)
	}
	items := slots.CatalogItems(res.Data)
	row, err := slots.PickCombination(items, query)
	if err != nil {
		return catalogMatch{}, err
	}
	label := catalogStringValue(row["name"])
	return catalogMatch{Type: catalogTypeCombination, Label: label, Raw: row}, nil
}

func matchIndex(ctx context.Context, query string, toolCtx tools.Context, runTool ToolRunner) (catalogMatch, error) {
	res := runTool(ctx, tools.CallRequest{Name: "get_index_signals", Arguments: map[string]any{}}, toolCtx)
	if res.Status != tools.StatusOK {
		return catalogMatch{}, fmt.Errorf("get_index_signals: %s", res.Summary)
	}
	items := slots.CatalogItems(res.Data)
	row, err := pickCatalogRow(items, query, "name", "index")
	if err != nil {
		return catalogMatch{}, err
	}
	label := catalogStringValue(row["name"])
	if label == "" {
		label = catalogStringValue(row["index"])
	}
	return catalogMatch{Type: catalogTypeIndex, Label: label, Raw: row}, nil
}

func matchCustomSkill(ctx context.Context, query string, toolCtx tools.Context, runTool ToolRunner) (catalogMatch, error) {
	if strings.TrimSpace(toolCtx.MCPToken) == "" {
		return catalogMatch{}, fmt.Errorf("custom strategies require mcp_token")
	}
	res := runTool(ctx, tools.CallRequest{Name: "get_custom_signal_for_skill", Arguments: map[string]any{}}, toolCtx)
	if res.Status != tools.StatusOK {
		return catalogMatch{}, fmt.Errorf("get_custom_signal_for_skill: %s", res.Summary)
	}
	items := slots.CatalogItems(res.Data)
	row, err := pickCatalogRow(items, query, "name", "brief")
	if err != nil {
		return catalogMatch{}, err
	}
	label := catalogStringValue(row["name"])
	return catalogMatch{Type: catalogTypeCustom, Label: label, Raw: row}, nil
}

func matchDefinition(ctx context.Context, query string, toolCtx tools.Context, runTool ToolRunner) (catalogMatch, error) {
	res := runTool(ctx, tools.CallRequest{Name: "get_custom_strategy_definitions", Arguments: map[string]any{}}, toolCtx)
	if res.Status != tools.StatusOK {
		return catalogMatch{}, fmt.Errorf("get_custom_strategy_definitions: %s", res.Summary)
	}
	items := slots.CatalogItems(res.Data)
	row, err := pickCatalogRow(items, query, "strategy_key", "name", "display_name", "title")
	if err != nil {
		return catalogMatch{}, err
	}
	label := firstNonEmpty(row, "display_name", "name", "title", "strategy_key")
	return catalogMatch{Type: catalogTypeDefinition, Label: label, Raw: row}, nil
}

func pickCatalogRow(items []map[string]any, query string, fields ...string) (map[string]any, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("empty catalog")
	}
	tokens := slotsSignalTokens(query)
	var matches []map[string]any
	for _, row := range items {
		if rowMatchesTokens(row, tokens, fields...) {
			matches = append(matches, row)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) == 0 {
		if len(items) == 1 && len(tokens) == 0 {
			return items[0], nil
		}
		return nil, fmt.Errorf("未找到匹配「%s」的策略", query)
	}
	return bestCatalogMatch(matches, tokens, fields...), nil
}

func rowMatchesTokens(row map[string]any, tokens []string, fields ...string) bool {
	if len(tokens) == 0 {
		return false
	}
	blob := strings.ToUpper(catalogRowText(row, fields...))
	for _, tok := range tokens {
		if !strings.Contains(blob, tok) {
			return false
		}
	}
	return true
}

func bestCatalogMatch(matches []map[string]any, tokens []string, fields ...string) map[string]any {
	best := matches[0]
	bestScore := scoreCatalogRow(best, tokens, fields...)
	for _, row := range matches[1:] {
		if score := scoreCatalogRow(row, tokens, fields...); score > bestScore {
			best = row
			bestScore = score
		}
	}
	return best
}

func scoreCatalogRow(row map[string]any, tokens []string, fields ...string) int {
	text := strings.ToUpper(catalogRowText(row, fields...))
	score := 0
	for _, tok := range tokens {
		if strings.Contains(text, tok) {
			score += 10
		}
	}
	score -= len([]rune(text)) / 20
	return score
}

func catalogRowText(row map[string]any, fields ...string) string {
	var parts []string
	for _, f := range fields {
		if v := catalogStringValue(row[f]); v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, " ")
}

func firstNonEmpty(row map[string]any, keys ...string) string {
	for _, k := range keys {
		if v := catalogStringValue(row[k]); v != "" {
			return v
		}
	}
	return ""
}

// catalogStringValue normalizes catalog fields; i18n name maps prefer cn → en.
func catalogStringValue(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if s == "" || s == "<nil>" {
			return ""
		}
		return s
	case map[string]any:
		return localizedCatalogString(t)
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" || s == "<nil>" || strings.HasPrefix(s, "map[") {
			return ""
		}
		return s
	}
}

func localizedCatalogString(m map[string]any) string {
	for _, key := range []string{"cn", "zh_cn", "zh", "hk", "en", "name", "title"} {
		if s := catalogStringValue(m[key]); s != "" {
			return s
		}
	}
	for _, v := range m {
		if s := catalogStringValue(v); s != "" {
			return s
		}
	}
	return ""
}

func slotsSignalTokens(query string) []string {
	parts := strings.Fields(strings.ToUpper(strings.NewReplacer(
		"策略认知", " ", "策略开发", " ", "学习", " ", "了解", " ", "知识库", " ",
	).Replace(query)))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || p == "策略" || p == "信号" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func catalogJSON(raw map[string]any) string {
	if raw == nil {
		return "{}"
	}
	b, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Sprint(raw)
	}
	return string(b)
}
