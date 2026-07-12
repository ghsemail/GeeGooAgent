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
}

// QueryConfigured posts {"type":"configured"} to baseURL/queryModel.
func QueryConfigured(ctx context.Context, baseURL string) (ConfiguredModel, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return ConfiguredModel{}, fmt.Errorf("empty admin/catalog base url")
	}
	body, _ := json.Marshal(map[string]string{"type": "configured"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/queryModel", bytes.NewReader(body))
	if err != nil {
		return ConfiguredModel{}, err
	}
	req.Header.Set("Content-Type", "application/json")
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

// QueryConfiguredFromCandidates tries each base URL until one succeeds.
func QueryConfiguredFromCandidates(ctx context.Context, bases ...string) (ConfiguredModel, string, error) {
	var last error
	for _, b := range bases {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		doc, err := QueryConfigured(ctx, b)
		if err == nil {
			return doc, b, nil
		}
		last = err
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
