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

// ModelRuntimeConfigView mirrors catalog getModelRuntimeConfig response data.
type ModelRuntimeConfigView struct {
	ConfiguredModelID string            `json:"configured_model_id"`
	ConfiguredModel   *ConfiguredModel  `json:"configured_model"`
	FallbackModelIDs  []string          `json:"fallback_model_ids"`
	FallbackModels    []ConfiguredModel `json:"fallback_models"`
	UpdatedAt         string            `json:"updated_at"`
}

type modelRuntimeConfigResponse struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    ModelRuntimeConfigView `json:"data"`
}

// QueryModelRuntimeConfigFromTargets reads ops fallback chain from catalog-api.
func QueryModelRuntimeConfigFromTargets(ctx context.Context, targets []QueryTarget) (ModelRuntimeConfigView, error) {
	var last error
	for _, t := range targets {
		base := strings.TrimSpace(t.BaseURL)
		if base == "" {
			continue
		}
		view, err := queryModelRuntimeConfig(ctx, base, t.Bearer)
		if err != nil {
			last = err
			continue
		}
		return view, nil
	}
	if last == nil {
		last = fmt.Errorf("no catalog endpoints configured")
	}
	return ModelRuntimeConfigView{}, last
}

func queryModelRuntimeConfig(ctx context.Context, baseURL, bearer string) (ModelRuntimeConfigView, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return ModelRuntimeConfigView{}, fmt.Errorf("empty admin/catalog base url")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/getModelRuntimeConfig", bytes.NewReader([]byte("{}")))
	if err != nil {
		return ModelRuntimeConfigView{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(bearer) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(bearer))
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ModelRuntimeConfigView{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return ModelRuntimeConfigView{}, fmt.Errorf("getModelRuntimeConfig HTTP %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	var envelope modelRuntimeConfigResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return ModelRuntimeConfigView{}, fmt.Errorf("decode getModelRuntimeConfig: %w", err)
	}
	if envelope.Code != 100 {
		msg := strings.TrimSpace(envelope.Message)
		if msg == "" {
			msg = "getModelRuntimeConfig failed"
		}
		return ModelRuntimeConfigView{}, fmt.Errorf("%s", msg)
	}
	return envelope.Data, nil
}
