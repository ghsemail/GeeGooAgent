package admin

import (
	"context"
	"fmt"
	"strings"
)

// ListModelsFromTargets calls getModel on the first configured catalog target.
func ListModelsFromTargets(ctx context.Context, targets []QueryTarget) ([]ConfiguredModel, string, error) {
	t, err := firstCatalogTarget(targets)
	if err != nil {
		return nil, "", err
	}
	docs, err := ListModels(ctx, t.BaseURL, t.Bearer)
	if err != nil {
		return nil, t.BaseURL, err
	}
	return docs, t.BaseURL, nil
}

// QueryModelFromTargets resolves one catalog row by model_id or configured default.
func QueryModelFromTargets(ctx context.Context, targets []QueryTarget, modelID string, useConfigured bool) (ConfiguredModel, string, error) {
	t, err := firstCatalogTarget(targets)
	if err != nil {
		return ConfiguredModel{}, "", err
	}
	var doc ConfiguredModel
	switch {
	case strings.TrimSpace(modelID) != "":
		doc, err = QueryModelByID(ctx, t.BaseURL, t.Bearer, modelID)
	case useConfigured:
		doc, err = QueryConfiguredWithBearer(ctx, t.BaseURL, t.Bearer)
	default:
		return ConfiguredModel{}, "", fmt.Errorf("no catalog model selected")
	}
	if err != nil {
		return ConfiguredModel{}, t.BaseURL, err
	}
	return doc, t.BaseURL, nil
}

func firstCatalogTarget(targets []QueryTarget) (QueryTarget, error) {
	for _, t := range targets {
		if strings.TrimSpace(t.BaseURL) != "" {
			return t, nil
		}
	}
	return QueryTarget{}, fmt.Errorf("no catalog query targets configured")
}
