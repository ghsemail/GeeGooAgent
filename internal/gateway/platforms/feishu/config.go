package feishu

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/gateway"
)

// Config holds Feishu/Lark credentials and policies (M1).
type Config struct {
	AppID          string
	AppSecret      string
	Domain         string // feishu | lark
	ConnectionMode string // websocket (M1); webhook reserved
	AllowedUsers   []string
	AllowAllUsers  bool
	HomeChannel    string
	GroupPolicy    string // allowlist | open | disabled
	RequireMention bool
	BotOpenID      string // optional override
	BotName        string
	ToolProgress   bool // editable progress bubble (default true)
}

// LoadConfigFromEnv reads FEISHU_* environment variables.
func LoadConfigFromEnv(getenv func(string) string) Config {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	requireMention := true
	if v := strings.TrimSpace(strings.ToLower(getenv("FEISHU_REQUIRE_MENTION"))); v == "false" || v == "0" {
		requireMention = false
	}
	domain := strings.TrimSpace(strings.ToLower(getenv("FEISHU_DOMAIN")))
	if domain == "" {
		domain = "feishu"
	}
	mode := strings.TrimSpace(strings.ToLower(getenv("FEISHU_CONNECTION_MODE")))
	if mode == "" {
		mode = "websocket"
	}
	policy := strings.TrimSpace(strings.ToLower(getenv("FEISHU_GROUP_POLICY")))
	if policy == "" {
		policy = "allowlist"
	}
	allowAll := strings.EqualFold(strings.TrimSpace(getenv("FEISHU_ALLOW_ALL_USERS")), "true")
	toolProgress := true
	if v := strings.TrimSpace(strings.ToLower(getenv("FEISHU_TOOL_PROGRESS"))); v == "0" || v == "false" || v == "off" || v == "none" {
		toolProgress = false
	}
	return Config{
		AppID:          strings.TrimSpace(getenv("FEISHU_APP_ID")),
		AppSecret:      strings.TrimSpace(getenv("FEISHU_APP_SECRET")),
		Domain:         domain,
		ConnectionMode: mode,
		AllowedUsers:   splitCSV(getenv("FEISHU_ALLOWED_USERS")),
		AllowAllUsers:  allowAll,
		HomeChannel:    strings.TrimSpace(getenv("FEISHU_HOME_CHANNEL")),
		GroupPolicy:    policy,
		RequireMention: requireMention,
		BotOpenID:      strings.TrimSpace(getenv("FEISHU_BOT_OPEN_ID")),
		BotName:        strings.TrimSpace(getenv("FEISHU_BOT_NAME")),
		ToolProgress:   toolProgress,
	}
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// AllowedSet returns a set for gateway.Config.
func (c Config) AllowedSet() map[string]struct{} {
	m := make(map[string]struct{}, len(c.AllowedUsers))
	for _, u := range c.AllowedUsers {
		m[u] = struct{}{}
	}
	return m
}

// GatewayAllowAll reports whether the runner should skip allowlist checks.
func (c Config) GatewayAllowAll() bool {
	return c.AllowAllUsers || len(c.AllowedUsers) == 0
}

// Home returns optional home channel for M2.
func (c Config) Home() (gateway.HomeChannel, bool) {
	if c.HomeChannel == "" {
		return gateway.HomeChannel{}, false
	}
	return gateway.HomeChannel{Platform: gateway.PlatformFeishu, ChatID: c.HomeChannel, Name: "Home"}, true
}
