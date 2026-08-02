package admin

import (
	"context"
	"fmt"
	"strings"
)

func normalizeModelAlias(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, " ", "")
	return s
}

func modelMatchesAlias(m ConfiguredModel, alias string) bool {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return false
	}
	if strings.EqualFold(m.ModelID, alias) {
		return true
	}
	for _, candidate := range []string{m.Name, m.DisplayName} {
		c := strings.TrimSpace(candidate)
		if c == "" {
			continue
		}
		if strings.EqualFold(c, alias) {
			return true
		}
		if normalizeModelAlias(c) == normalizeModelAlias(alias) {
			return true
		}
	}
	return false
}

func resolveCatalogModelID(ctx context.Context, targets []QueryTarget, alias string) (string, error) {
	docs, _, err := ListModelsFromTargets(ctx, targets)
	if err != nil {
		return "", err
	}
	for _, d := range docs {
		if modelMatchesAlias(d, alias) && strings.TrimSpace(d.ModelID) != "" {
			return strings.TrimSpace(d.ModelID), nil
		}
	}
	return "", fmt.Errorf("catalog model %q not found", alias)
}

func queryModelFromTargetsResolved(ctx context.Context, targets []QueryTarget, modelID string) (ConfiguredModel, string, error) {
	t, err := firstCatalogTarget(targets)
	if err != nil {
		return ConfiguredModel{}, "", err
	}
	id := strings.TrimSpace(modelID)
	if id == "" {
		return ConfiguredModel{}, "", fmt.Errorf("empty catalog model id")
	}
	doc, err := QueryModelByID(ctx, t.BaseURL, t.Bearer, id)
	if err == nil {
		return doc, t.BaseURL, nil
	}
	resolved, listErr := resolveCatalogModelID(ctx, targets, id)
	if listErr != nil {
		return ConfiguredModel{}, t.BaseURL, err
	}
	doc, err = QueryModelByID(ctx, t.BaseURL, t.Bearer, resolved)
	if err != nil {
		return ConfiguredModel{}, t.BaseURL, err
	}
	return doc, t.BaseURL, nil
}
