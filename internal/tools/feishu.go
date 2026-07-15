package tools

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

func postFeishuText(ctx context.Context, webhookURL, text string) error {
	webhookURL = strings.TrimSpace(webhookURL)
	text = strings.TrimSpace(text)
	if webhookURL == "" {
		return fmt.Errorf("feishu webhook not configured")
	}
	if text == "" {
		return fmt.Errorf("feishu message text is empty")
	}
	body, err := json.Marshal(map[string]any{
		"msg_type": "text",
		"content":  map[string]string{"text": text},
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("feishu webhook HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var parsed map[string]any
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &parsed)
	}
	if code, ok := parsed["code"].(float64); ok && int(code) != 0 {
		msg, _ := parsed["msg"].(string)
		if msg == "" {
			msg = string(raw)
		}
		return fmt.Errorf("feishu webhook rejected: %s", msg)
	}
	return nil
}
