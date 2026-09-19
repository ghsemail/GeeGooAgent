package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

const (
	StrategyCatalogCombination = "combination"
	StrategyCatalogDefinition  = "definition"
	StrategyCatalogCustom      = "custom"
)

// StrategyCatalogEntry is one selectable strategy for workflow cognition eval.
type StrategyCatalogEntry struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Label string `json:"label"`
}

// ResolveGenerateCognitionMessage builds the Dock Chat utterance for generate_strategy_cognition eval.
func ResolveGenerateCognitionMessage(opts TurnPlanCaseOptions, pickedName string) string {
	name := strings.TrimSpace(pickedName)
	if name == "" {
		name = strings.TrimSpace(opts.StrategyName)
	}
	if name == "" {
		msg := strings.TrimSpace(opts.Message)
		if msg != "" {
			return msg
		}
		return formatGenerateStrategyArchiveMessage("")
	}
	return formatGenerateStrategyArchiveMessage(name)
}

func formatGenerateStrategyArchiveMessage(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "帮我生成策略档案"
	}
	return fmt.Sprintf("帮我生成 %s 的策略档案", name)
}

// PickRandomStrategyEntry picks one strategy from enabled catalog types (combination, definition, custom).
func PickRandomStrategyEntry(ctx context.Context, catalogURL, apiKey, mcpToken string, allowedTypes []string) (StrategyCatalogEntry, error) {
	entries, err := ListStrategyCatalog(ctx, catalogURL, apiKey, mcpToken, allowedTypes)
	if err != nil {
		return StrategyCatalogEntry{}, err
	}
	if len(entries) == 0 {
		return StrategyCatalogEntry{}, fmt.Errorf("no strategies in catalog")
	}
	picked := entries[rand.Intn(len(entries))]
	return picked, nil
}

// ListStrategyCatalog returns strategies for eval UI / random pick.
func ListStrategyCatalog(ctx context.Context, catalogURL, apiKey, mcpToken string, types []string) ([]StrategyCatalogEntry, error) {
	base := strings.TrimRight(strings.TrimSpace(catalogURL), "/")
	if base == "" {
		return nil, fmt.Errorf("signal catalog url not configured")
	}
	want := normalizeStrategyCatalogTypes(types, mcpToken)
	out := make([]StrategyCatalogEntry, 0, 64)
	for _, typ := range want {
		rows, err := fetchStrategyCatalogRows(ctx, base, apiKey, mcpToken, typ)
		if err != nil {
			if typ == StrategyCatalogCustom {
				continue
			}
			return nil, err
		}
		for _, row := range rows {
			label := strategyCatalogLabel(row, typ)
			if label == "" {
				continue
			}
			out = append(out, StrategyCatalogEntry{Type: typ, Name: label, Label: label})
		}
	}
	return out, nil
}

func normalizeStrategyCatalogTypes(types []string, mcpToken string) []string {
	if len(types) == 0 {
		out := []string{StrategyCatalogCombination, StrategyCatalogDefinition}
		if strings.TrimSpace(mcpToken) != "" {
			out = append(out, StrategyCatalogCustom)
		}
		return out
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(types))
	for _, raw := range types {
		typ := strings.ToLower(strings.TrimSpace(raw))
		switch typ {
		case StrategyCatalogCombination, StrategyCatalogDefinition, StrategyCatalogCustom:
			if typ == StrategyCatalogCustom && strings.TrimSpace(mcpToken) == "" {
				continue
			}
			if _, ok := seen[typ]; ok {
				continue
			}
			seen[typ] = struct{}{}
			out = append(out, typ)
		}
	}
	if len(out) == 0 {
		return []string{StrategyCatalogCombination, StrategyCatalogDefinition, StrategyCatalogCustom}
	}
	return out
}

func fetchStrategyCatalogRows(ctx context.Context, base, apiKey, mcpToken, catalogType string) ([]map[string]any, error) {
	path := ""
	requiresMCP := false
	switch catalogType {
	case StrategyCatalogCombination:
		path = "/getSignalCombinationForSkill"
	case StrategyCatalogDefinition:
		path = "/getCustomStrategyDefinitions"
	case StrategyCatalogCustom:
		path = "/getCustomSignalForSkill"
		requiresMCP = true
	default:
		return nil, fmt.Errorf("unsupported strategy catalog type: %s", catalogType)
	}
	if requiresMCP && strings.TrimSpace(mcpToken) == "" {
		return nil, fmt.Errorf("custom strategies require mcp_token")
	}
	raw, err := postCatalogJSON(ctx, base+path, apiKey, mcpToken, requiresMCP)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", catalogType, err)
	}
	return catalogItems(raw), nil
}

func postCatalogJSON(ctx context.Context, url, apiKey, mcpToken string, useMCP bool) ([]byte, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, strings.NewReader("{}"))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	key := strings.TrimSpace(apiKey)
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("X-API-Key", key)
	}
	if useMCP {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(mcpToken))
		req.Header.Set("X-API-Key", strings.TrimSpace(mcpToken))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncateEvalText(string(raw), 400))
	}
	return raw, nil
}

func strategyCatalogLabel(row map[string]any, catalogType string) string {
	switch catalogType {
	case StrategyCatalogDefinition:
		return firstCatalogString(row, "display_name", "name", "title", "strategy_key")
	default:
		return firstCatalogString(row, "name", "display_name", "title", "strategy_key", "index")
	}
}

func firstCatalogString(row map[string]any, keys ...string) string {
	for _, key := range keys {
		raw, ok := row[key]
		if !ok || raw == nil {
			continue
		}
		switch v := raw.(type) {
		case string:
			if s := strings.TrimSpace(v); s != "" {
				return s
			}
		case map[string]any:
			for _, lang := range []string{"cn", "zh", "en", "hk"} {
				if s := strings.TrimSpace(fmt.Sprint(v[lang])); s != "" && s != "<nil>" {
					return s
				}
			}
		default:
			if s := strings.TrimSpace(fmt.Sprint(v)); s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

// PickRandomStrategyName loads combination strategies from GeeGooSignal catalog-api and picks one.
// Deprecated: use PickRandomStrategyEntry for multi-type random eval.
func PickRandomStrategyName(ctx context.Context, catalogURL, apiKey string) (string, error) {
	entry, err := PickRandomStrategyEntry(ctx, catalogURL, apiKey, "", []string{StrategyCatalogCombination})
	if err != nil {
		return "", err
	}
	return entry.Name, nil
}

func strategyNamesFromCatalogPayload(raw []byte) ([]string, error) {
	items := catalogItems(raw)
	names := make([]string, 0, len(items))
	for _, row := range items {
		name := strategyCatalogLabel(row, StrategyCatalogCombination)
		if name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

func catalogItems(data any) []map[string]any {
	switch v := data.(type) {
	case []byte:
		var parsed any
		if err := json.Unmarshal(v, &parsed); err != nil {
			return nil
		}
		return catalogItems(parsed)
	case []any:
		return asMapSlice(v)
	case map[string]any:
		for _, key := range []string{"data", "items", "value"} {
			if nested, ok := v[key]; ok {
				if out := catalogItems(nested); len(out) > 0 {
					return out
				}
			}
		}
		if nested, ok := v["items"].(map[string]any); ok {
			for _, key := range []string{"items", "data"} {
				if out := catalogItems(nested[key]); len(out) > 0 {
					return out
				}
			}
		}
	}
	return nil
}

func truncateEvalText(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max]
}

func asMapSlice(items []any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if ok {
			out = append(out, m)
		}
	}
	return out
}
