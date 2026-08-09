package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	registrationPath     = "/oauth/v1/app/registration"
	onboardRequestTimeout = 15 * time.Second
)

var (
	onboardAccountsURLs = map[string]string{
		"feishu": "https://accounts.feishu.cn",
		"lark":   "https://accounts.larksuite.com",
	}
	onboardOpenURLs = map[string]string{
		"feishu": "https://open.feishu.cn",
		"lark":   "https://open.larksuite.com",
	}
)

// Credentials is the result of QR scan-to-create onboarding.
type Credentials struct {
	AppID     string
	AppSecret string
	Domain    string // feishu | lark
	OpenID    string // scanning user's open_id (optional)
	BotName   string
	BotOpenID string
}

// OnboardHTTPClient is overridable in tests.
var OnboardHTTPClient = &http.Client{Timeout: onboardRequestTimeout}

// BeginResult is returned by the registration "begin" action.
type BeginResult struct {
	DeviceCode string
	QRURL      string
	UserCode   string
	Interval   int
	ExpireIn   int
}

func accountsBaseURL(domain string) string {
	if u, ok := onboardAccountsURLs[strings.ToLower(domain)]; ok {
		return u
	}
	return onboardAccountsURLs["feishu"]
}

func openBaseURL(domain string) string {
	if u, ok := onboardOpenURLs[strings.ToLower(domain)]; ok {
		return u
	}
	return onboardOpenURLs["feishu"]
}

func postRegistration(ctx context.Context, baseURL string, body map[string]string) (map[string]any, error) {
	form := url.Values{}
	for k, v := range body {
		form.Set(k, v)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+registrationPath, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := OnboardHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// Endpoint returns JSON even on 4xx (e.g. authorization_pending).
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("feishu registration HTTP %d: %s", resp.StatusCode, truncateBytes(raw, 200))
		}
		return nil, fmt.Errorf("feishu registration: decode: %w", err)
	}
	return out, nil
}

func truncateBytes(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}

// InitRegistration checks the environment supports client_secret auth.
func InitRegistration(ctx context.Context, domain string) error {
	res, err := postRegistration(ctx, accountsBaseURL(domain), map[string]string{"action": "init"})
	if err != nil {
		return err
	}
	methods, _ := res["supported_auth_methods"].([]any)
	for _, m := range methods {
		if s, ok := m.(string); ok && s == "client_secret" {
			return nil
		}
	}
	return fmt.Errorf("feishu registration does not support client_secret auth (got %v)", methods)
}

// BeginRegistration starts the device-code flow and returns the QR URL.
func BeginRegistration(ctx context.Context, domain string) (*BeginResult, error) {
	res, err := postRegistration(ctx, accountsBaseURL(domain), map[string]string{
		"action":            "begin",
		"archetype":         "PersonalAgent",
		"auth_method":       "client_secret",
		"request_user_info": "open_id",
	})
	if err != nil {
		return nil, err
	}
	deviceCode, _ := res["device_code"].(string)
	if deviceCode == "" {
		return nil, fmt.Errorf("feishu registration: missing device_code")
	}
	qrURL, _ := res["verification_uri_complete"].(string)
	if qrURL == "" {
		return nil, fmt.Errorf("feishu registration: missing verification_uri_complete")
	}
	if strings.Contains(qrURL, "?") {
		qrURL += "&from=geegoo&tp=geegoo"
	} else {
		qrURL += "?from=geegoo&tp=geegoo"
	}
	interval := asInt(res["interval"], 5)
	expire := asInt(res["expire_in"], 600)
	userCode, _ := res["user_code"].(string)
	return &BeginResult{
		DeviceCode: deviceCode,
		QRURL:      qrURL,
		UserCode:   userCode,
		Interval:   interval,
		ExpireIn:   expire,
	}, nil
}

// PollRegistration waits until the user scans the QR and the bot app is created.
func PollRegistration(ctx context.Context, deviceCode string, interval, expireIn int, domain string) (*Credentials, error) {
	if interval < 1 {
		interval = 5
	}
	if expireIn < 1 {
		expireIn = 600
	}
	deadline := time.Now().Add(time.Duration(expireIn) * time.Second)
	currentDomain := domain
	domainSwitched := false

	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		res, err := postRegistration(ctx, accountsBaseURL(currentDomain), map[string]string{
			"action":      "poll",
			"device_code": deviceCode,
			"tp":          "ob_app",
		})
		if err != nil {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(interval) * time.Second):
			}
			continue
		}

		if ui, ok := res["user_info"].(map[string]any); ok {
			if brand, _ := ui["tenant_brand"].(string); brand == "lark" && !domainSwitched {
				currentDomain = "lark"
				domainSwitched = true
			}
		}

		clientID, _ := res["client_id"].(string)
		clientSecret, _ := res["client_secret"].(string)
		if clientID != "" && clientSecret != "" {
			openID := ""
			if ui, ok := res["user_info"].(map[string]any); ok {
				openID, _ = ui["open_id"].(string)
			}
			return &Credentials{
				AppID:     clientID,
				AppSecret: clientSecret,
				Domain:    currentDomain,
				OpenID:    openID,
			}, nil
		}

		errCode, _ := res["error"].(string)
		switch errCode {
		case "access_denied", "expired_token":
			return nil, fmt.Errorf("feishu registration %s", errCode)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(interval) * time.Second):
		}
	}
	return nil, fmt.Errorf("feishu registration timed out after %ds", expireIn)
}

// PollOnce performs one registration poll. status is "completed", "pending", or "failed".
func PollOnce(ctx context.Context, deviceCode, domain string) (creds *Credentials, status string, err error) {
	res, err := postRegistration(ctx, accountsBaseURL(domain), map[string]string{
		"action":      "poll",
		"device_code": deviceCode,
		"tp":          "ob_app",
	})
	if err != nil {
		return nil, "pending", nil
	}
	domainOut := domain
	if ui, ok := res["user_info"].(map[string]any); ok {
		if brand, _ := ui["tenant_brand"].(string); brand == "lark" {
			domainOut = "lark"
		}
	}
	clientID, _ := res["client_id"].(string)
	clientSecret, _ := res["client_secret"].(string)
	if clientID != "" && clientSecret != "" {
		openID := ""
		if ui, ok := res["user_info"].(map[string]any); ok {
			openID, _ = ui["open_id"].(string)
		}
		return &Credentials{
			AppID:     clientID,
			AppSecret: clientSecret,
			Domain:    domainOut,
			OpenID:    openID,
		}, "completed", nil
	}
	errCode, _ := res["error"].(string)
	switch errCode {
	case "access_denied", "expired_token":
		return nil, "failed", fmt.Errorf("feishu registration %s", errCode)
	default:
		return nil, "pending", nil
	}
}

func asInt(v any, def int) int {
	switch t := v.(type) {
	case float64:
		if int(t) > 0 {
			return int(t)
		}
	case int:
		if t > 0 {
			return t
		}
	case json.Number:
		if n, err := t.Int64(); err == nil && n > 0 {
			return int(n)
		}
	case string:
		var n int
		if _, err := fmt.Sscanf(t, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// ProbeBot verifies credentials via /open-apis/bot/v3/info.
func ProbeBot(ctx context.Context, appID, appSecret, domain string) (botName, botOpenID string, err error) {
	base := openBaseURL(domain)
	tokenBody, _ := json.Marshal(map[string]string{
		"app_id":     appID,
		"app_secret": appSecret,
	})
	tokenReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/open-apis/auth/v3/tenant_access_token/internal", strings.NewReader(string(tokenBody)))
	if err != nil {
		return "", "", err
	}
	tokenReq.Header.Set("Content-Type", "application/json")
	tokenResp, err := OnboardHTTPClient.Do(tokenReq)
	if err != nil {
		return "", "", err
	}
	defer tokenResp.Body.Close()
	tokenRaw, _ := io.ReadAll(tokenResp.Body)
	var tokenRes struct {
		TenantAccessToken string `json:"tenant_access_token"`
	}
	_ = json.Unmarshal(tokenRaw, &tokenRes)
	if tokenRes.TenantAccessToken == "" {
		return "", "", fmt.Errorf("feishu: no tenant_access_token")
	}

	botReq, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/open-apis/bot/v3/info", nil)
	if err != nil {
		return "", "", err
	}
	botReq.Header.Set("Authorization", "Bearer "+tokenRes.TenantAccessToken)
	botResp, err := OnboardHTTPClient.Do(botReq)
	if err != nil {
		return "", "", err
	}
	defer botResp.Body.Close()
	botRaw, _ := io.ReadAll(botResp.Body)
	var botRes struct {
		Code int `json:"code"`
		Bot  struct {
			OpenID  string `json:"open_id"`
			AppName string `json:"app_name"`
		} `json:"bot"`
	}
	if err := json.Unmarshal(botRaw, &botRes); err != nil {
		return "", "", err
	}
	if botRes.Code != 0 {
		return "", "", fmt.Errorf("feishu bot/v3/info code=%d", botRes.Code)
	}
	return botRes.Bot.AppName, botRes.Bot.OpenID, nil
}
