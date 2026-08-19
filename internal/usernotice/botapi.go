package usernotice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/gateway/feishustore"
)

func serviceAPIEnabled(cfg *config.AppConfig) bool {
	if cfg == nil {
		return false
	}
	return strings.TrimSpace(cfg.BotServiceAPIURL) != "" && strings.TrimSpace(cfg.BotServiceAPIKey) != ""
}

func serviceAPIURL(cfg *config.AppConfig) string {
	return strings.TrimRight(strings.TrimSpace(cfg.BotServiceAPIURL), "/")
}

func serviceAPIKey(cfg *config.AppConfig) string {
	return strings.TrimSpace(cfg.BotServiceAPIKey)
}

func loadCredsHTTP(ctx context.Context, cfg *config.AppConfig, userID string) (*feishustore.Creds, error) {
	body, _ := json.Marshal(map[string]string{"user_id": userID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serviceAPIURL(cfg)+"/getUserFeishuNotice", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+serviceAPIKey(cfg))
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
		return nil, fmt.Errorf("getUserFeishuNotice: %s", strings.TrimSpace(string(raw)))
	}
	var envelope struct {
		Feishu FeishuSection `json:"feishu"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	uid := strings.TrimSpace(envelope.Feishu.UserID)
	if uid == "" {
		uid = userID
	}
	return sectionToCreds(uid, envelope.Feishu), nil
}

func listCredsHTTP(ctx context.Context, cfg *config.AppConfig) ([]feishustore.Creds, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serviceAPIURL(cfg)+"/listFeishuGatewayUsers", bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+serviceAPIKey(cfg))
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
		return nil, fmt.Errorf("listFeishuGatewayUsers: %s", strings.TrimSpace(string(raw)))
	}
	var envelope struct {
		Users []FeishuSection `json:"users"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	out := make([]feishustore.Creds, 0, len(envelope.Users))
	for _, section := range envelope.Users {
		uid := strings.TrimSpace(section.UserID)
		if uid == "" {
			continue
		}
		c := sectionToCreds(uid, section)
		if c == nil || !c.Configured() {
			continue
		}
		out = append(out, *c)
	}
	return out, nil
}

func syncFeishuHTTP(ctx context.Context, cfg *config.AppConfig, userID string, creds *feishustore.Creds) error {
	body, _ := json.Marshal(map[string]any{
		"user_id": userID,
		"feishu":  credsToSection(creds),
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serviceAPIURL(cfg)+"/setUserFeishuNotice", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+serviceAPIKey(cfg))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("setUserFeishuNotice: %s", strings.TrimSpace(string(raw)))
	}
	return nil
}
