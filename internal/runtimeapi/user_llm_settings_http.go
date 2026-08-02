package runtimeapi

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

func (h *Handler) botServiceAPIURL() string {
	if h.App == nil || h.App.Config == nil {
		return ""
	}
	return strings.TrimRight(strings.TrimSpace(h.App.Config.BotServiceAPIURL), "/")
}

func (h *Handler) botServiceAPIKey() string {
	if h.App == nil || h.App.Config == nil {
		return ""
	}
	return strings.TrimSpace(h.App.Config.BotServiceAPIKey)
}

func (h *Handler) botServiceAPIEnabled() bool {
	return h.botServiceAPIURL() != "" && h.botServiceAPIKey() != ""
}

func (h *Handler) loadUserLLMSettingsHTTP(ctx context.Context, userID string) (*userLLMSettings, error) {
	base := h.botServiceAPIURL()
	if base == "" {
		return nil, fmt.Errorf("bot service api not configured")
	}
	body, _ := json.Marshal(map[string]string{"user_id": userID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/getUserLLMSettings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.botServiceAPIKey())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getUserLLMSettings: %s", strings.TrimSpace(string(raw)))
	}
	var doc struct {
		LLMSettings *userLLMSettings `json:"llm_settings"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return doc.LLMSettings, nil
}

func (h *Handler) saveUserLLMSettingsHTTP(ctx context.Context, userID string, settings *userLLMSettings) error {
	if settings == nil {
		return nil
	}
	base := h.botServiceAPIURL()
	if base == "" {
		return fmt.Errorf("bot service api not configured")
	}
	settings.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	body, _ := json.Marshal(map[string]any{
		"user_id":      userID,
		"llm_settings": settings,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/setUserLLMSettings", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.botServiceAPIKey())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("setUserLLMSettings: %s", strings.TrimSpace(string(raw)))
	}
	return nil
}
