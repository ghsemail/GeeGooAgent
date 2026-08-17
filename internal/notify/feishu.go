// Package notify delivers scheduler summaries to Feishu.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/gateway"
	"github.com/ghsemail/GeeGooAgent/internal/gateway/platforms/feishu"
)

const feishuWebhookTimeout = 15 * time.Second

// FeishuSender posts plain-text/markdown summaries to Feishu.
type FeishuSender struct {
	WebhookURL string
	HTTP       *http.Client
}

// NewFeishuSender builds a sender. WebhookURL may be empty to use app credentials fallback.
func NewFeishuSender(webhookURL string) *FeishuSender {
	return &FeishuSender{
		WebhookURL: strings.TrimSpace(webhookURL),
		HTTP:       &http.Client{Timeout: feishuWebhookTimeout},
	}
}

// Send delivers text. Tries webhook first, then Feishu Open API (FEISHU_APP_ID/SECRET + chat target).
func (s *FeishuSender) Send(ctx context.Context, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("notify: empty message")
	}
	if s != nil && s.WebhookURL != "" {
		if err := s.sendWebhook(ctx, text); err == nil {
			return nil
		} else if !hasFeishuAppCreds() {
			return err
		}
	}
	return sendViaFeishuApp(ctx, text)
}

func (s *FeishuSender) sendWebhook(ctx context.Context, text string) error {
	body, err := json.Marshal(map[string]any{
		"msg_type": "text",
		"content":  map[string]string{"text": text},
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("feishu webhook HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var parsed struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if json.Unmarshal(raw, &parsed) == nil && parsed.Code != 0 {
		return fmt.Errorf("feishu webhook code=%d msg=%s", parsed.Code, parsed.Msg)
	}
	return nil
}

func hasFeishuAppCreds() bool {
	cfg := feishu.LoadConfigFromEnv(os.Getenv)
	return cfg.AppID != "" && cfg.AppSecret != ""
}

func sendViaFeishuApp(ctx context.Context, text string) error {
	cfg := feishu.LoadConfigFromEnv(os.Getenv)
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return fmt.Errorf("notify: feishu not configured (set feishu_webhook_url or FEISHU_APP_ID/SECRET)")
	}
	chatID := strings.TrimSpace(os.Getenv("GEEGOO_SCHEDULER_FEISHU_CHAT_ID"))
	if chatID == "" {
		chatID = cfg.HomeChannel
	}
	if chatID == "" && len(cfg.AllowedUsers) > 0 {
		// DM fallback: send to first allowlisted user open_id.
		chatID = cfg.AllowedUsers[0]
	}
	if chatID == "" {
		return fmt.Errorf("notify: missing FEISHU_HOME_CHANNEL / GEEGOO_SCHEDULER_FEISHU_CHAT_ID / FEISHU_ALLOWED_USERS")
	}
	ad := feishu.NewAdapter(cfg)
	if ad == nil || !ad.Configured() {
		return fmt.Errorf("notify: feishu adapter not configured")
	}
	return ad.SendText(ctx, gateway.OutboundText{ChatID: chatID, Text: text})
}
