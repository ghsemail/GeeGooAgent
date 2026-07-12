package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ConfiguredModel is the ops-configured LLM row from POST /queryModel.
type ConfiguredModel struct {
	ModelID     string `json:"model_id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
	Token       string `json:"token"`
	BaseURL     string `json:"base_url"`
	Provider    string `json:"provider,omitempty"`
}

// QueryConfigured posts {"type":"configured"} to baseURL/queryModel.
func QueryConfigured(ctx context.Context, baseURL string) (ConfiguredModel, error) {
	return QueryConfiguredWithBearer(ctx, baseURL, "")
}

// QueryModelByID posts {"model_id":...} to baseURL/queryModel.
func QueryModelByID(ctx context.Context, baseURL, bearer, modelID string) (ConfiguredModel, error) {
	return postQueryModel(ctx, baseURL, bearer, map[string]string{"model_id": strings.TrimSpace(modelID)})
}

// ListModels posts to baseURL/getModel and returns catalog rows.
func ListModels(ctx context.Context, baseURL, bearer string) ([]ConfiguredModel, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("empty admin/catalog base url")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/getModel", bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(bearer) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(bearer))
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getModel HTTP %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	var docs []ConfiguredModel
	if err := json.Unmarshal(raw, &docs); err != nil {
		return nil, fmt.Errorf("decode getModel: %w", err)
	}
	return docs, nil
}

// QueryConfiguredWithBearer posts queryModel with optional catalog-api Bearer.
func QueryConfiguredWithBearer(ctx context.Context, baseURL, bearer string) (ConfiguredModel, error) {
	return postQueryModel(ctx, baseURL, bearer, map[string]string{"type": "configured"})
}

func postQueryModel(ctx context.Context, baseURL, bearer string, body map[string]string) (ConfiguredModel, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return ConfiguredModel{}, fmt.Errorf("empty admin/catalog base url")
	}
	payload, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/queryModel", bytes.NewReader(payload))
	if err != nil {
		return ConfiguredModel{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(bearer) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(bearer))
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ConfiguredModel{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return ConfiguredModel{}, fmt.Errorf("queryModel HTTP %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	var doc ConfiguredModel
	if err := json.Unmarshal(raw, &doc); err != nil {
		return ConfiguredModel{}, fmt.Errorf("decode queryModel: %w", err)
	}
	if strings.TrimSpace(doc.Name) == "" && strings.TrimSpace(doc.DisplayName) == "" {
		return ConfiguredModel{}, fmt.Errorf("queryModel returned empty model")
	}
	return doc, nil
}

// QueryTarget is one queryModel endpoint with optional Bearer (catalog-api :3210).
type QueryTarget struct {
	BaseURL string
	Bearer  string
}

// QueryConfiguredFromCandidates tries each base URL until one succeeds with a non-empty token.
func QueryConfiguredFromCandidates(ctx context.Context, bases ...string) (ConfiguredModel, string, error) {
	targets := make([]QueryTarget, 0, len(bases))
	for _, b := range bases {
		targets = append(targets, QueryTarget{BaseURL: b})
	}
	return QueryConfiguredFromTargets(ctx, targets...)
}

// QueryConfiguredFromTargets tries each target; skips responses missing token (incomplete catalog rows).
func QueryConfiguredFromTargets(ctx context.Context, targets ...QueryTarget) (ConfiguredModel, string, error) {
	var last error
	for _, t := range targets {
		base := strings.TrimSpace(t.BaseURL)
		if base == "" {
			continue
		}
		doc, err := QueryConfiguredWithBearer(ctx, base, t.Bearer)
		if err != nil {
			last = err
			continue
		}
		if strings.TrimSpace(doc.Token) == "" {
			last = fmt.Errorf("queryModel at %s returned empty token", base)
			continue
		}
		return doc, base, nil
	}
	if last == nil {
		last = fmt.Errorf("no queryModel endpoints configured")
	}
	return ConfiguredModel{}, "", last
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
