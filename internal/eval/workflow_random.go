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

// PickRandomStrategyName loads combination strategies from GeeGooSignal catalog-api and picks one.
func PickRandomStrategyName(ctx context.Context, catalogURL, apiKey string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(catalogURL), "/")
	if base == "" {
		return "", fmt.Errorf("signal catalog url not configured")
	}
	reqCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, base+"/getSignalCombinationForSkill", strings.NewReader("{}"))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if key := strings.TrimSpace(apiKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("X-API-Key", key)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("catalog HTTP %d: %s", resp.StatusCode, truncateEvalText(string(raw), 400))
	}
	names, err := strategyNamesFromCatalogPayload(raw)
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "", fmt.Errorf("no combination strategies in catalog")
	}
	return names[rand.Intn(len(names))], nil
}

func strategyNamesFromCatalogPayload(raw []byte) ([]string, error) {
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	items := catalogItems(data)
	names := make([]string, 0, len(items))
	for _, row := range items {
		name := strings.TrimSpace(fmt.Sprint(row["name"]))
		if name != "" && name != "<nil>" {
			names = append(names, name)
		}
	}
	return names, nil
}

func catalogItems(data any) []map[string]any {
	switch v := data.(type) {
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

func truncateEvalText(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max]
}
