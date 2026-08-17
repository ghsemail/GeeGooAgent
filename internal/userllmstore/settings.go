package userllmstore

import (
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// Settings mirrors QT_DB.user.llm_settings (Agent Gateway model prefs).
type Settings struct {
	Provider       string                  `json:"provider,omitempty" bson:"provider,omitempty"`
	Model          string                  `json:"model,omitempty" bson:"model,omitempty"`
	CatalogModelID string                  `json:"catalog_model_id,omitempty" bson:"catalog_model_id,omitempty"`
	UseOpsModel    *bool                   `json:"use_ops_model,omitempty" bson:"use_ops_model,omitempty"`
	Thinking       string                  `json:"thinking,omitempty" bson:"thinking,omitempty"`
	Temperature    *float64                `json:"temperature,omitempty" bson:"temperature,omitempty"`
	MaxTokens      *int                    `json:"max_tokens,omitempty" bson:"max_tokens,omitempty"`
	Pinned         []string                `json:"pinned,omitempty" bson:"pinned,omitempty"`
	Gateways       map[string]GatewayEntry `json:"gateways,omitempty" bson:"gateways,omitempty"`
	UpdatedAt      string                  `json:"updated_at,omitempty" bson:"updated_at,omitempty"`
}

// GatewayEntry is per-channel model override (web | trading_app | feishu).
type GatewayEntry struct {
	Provider       string `json:"provider,omitempty" bson:"provider,omitempty"`
	Model          string `json:"model,omitempty" bson:"model,omitempty"`
	CatalogModelID string `json:"catalog_model_id,omitempty" bson:"catalog_model_id,omitempty"`
	UseOpsModel    *bool  `json:"use_ops_model,omitempty" bson:"use_ops_model,omitempty"`
}

// DashboardPatch is the subset of dashboard settings applied to user llm_settings.
type DashboardPatch struct {
	Provider       string
	Model          string
	CatalogModelID string
	Gateway        string
	UseOpsModel    *bool
	Thinking       string
	Temperature    *float64
	MaxTokens      *int
	Pinned         []string
}

// SanitizeUserID makes a user id safe for filesystem paths.
func SanitizeUserID(userID string) string {
	safe := safeUserID.ReplaceAllString(strings.TrimSpace(userID), "_")
	if safe == "" {
		return "anonymous"
	}
	return safe
}

// NormalizeGateway maps client channel ids to gateway keys.
func NormalizeGateway(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "", "web", "trading_operation":
		return "web"
	case "trading_app", "geegoo_agent", "geegoo-app", "app":
		return "trading_app"
	case "feishu", "lark":
		return "feishu"
	default:
		return s
	}
}

// ApplyDashboard merges a dashboard settings request into stored prefs.
func (u *Settings) ApplyDashboard(patch DashboardPatch) {
	if u == nil {
		return
	}
	gwKey := NormalizeGateway(patch.Gateway)
	if strings.TrimSpace(patch.Gateway) != "" && gwKey != "" {
		if u.Gateways == nil {
			u.Gateways = map[string]GatewayEntry{}
		}
		entry := u.Gateways[gwKey]
		entry.applyDashboard(patch)
		u.Gateways[gwKey] = entry
		return
	}
	if p := strings.TrimSpace(patch.Provider); p != "" {
		u.Provider = p
	}
	if m := strings.TrimSpace(patch.Model); m != "" {
		u.Model = m
	}
	if id := strings.TrimSpace(patch.CatalogModelID); id != "" {
		u.CatalogModelID = id
		useOps := false
		u.UseOpsModel = &useOps
	} else if patch.UseOpsModel != nil {
		u.UseOpsModel = patch.UseOpsModel
		if *patch.UseOpsModel {
			u.CatalogModelID = ""
		}
	}
	if t := strings.ToLower(strings.TrimSpace(patch.Thinking)); t != "" {
		u.Thinking = t
	}
	if patch.Temperature != nil && *patch.Temperature > 0 {
		u.Temperature = patch.Temperature
	}
	if patch.MaxTokens != nil && *patch.MaxTokens > 0 {
		u.MaxTokens = patch.MaxTokens
	}
	if len(patch.Pinned) > 0 {
		u.Pinned = append([]string(nil), patch.Pinned...)
	}
}

func (g *GatewayEntry) applyDashboard(patch DashboardPatch) {
	if g == nil {
		return
	}
	if p := strings.TrimSpace(patch.Provider); p != "" {
		g.Provider = p
	}
	if m := strings.TrimSpace(patch.Model); m != "" {
		g.Model = m
	}
	if id := strings.TrimSpace(patch.CatalogModelID); id != "" {
		g.CatalogModelID = id
		useOps := false
		g.UseOpsModel = &useOps
	} else if patch.UseOpsModel != nil {
		g.UseOpsModel = patch.UseOpsModel
		if *patch.UseOpsModel {
			g.CatalogModelID = ""
		}
	}
}

// MergeEffective overlays user + gateway prefs on base agent llm config.
func MergeEffective(base config.LLMConfig, doc *Settings, gateway string) config.LLMConfig {
	if doc == nil {
		return base
	}
	out := doc.mergeInto(base)
	gwKey := NormalizeGateway(gateway)
	if gwKey != "" && doc.Gateways != nil {
		if gw, ok := doc.Gateways[gwKey]; ok {
			out = gw.mergeInto(out)
		}
	}
	return out
}

func (g GatewayEntry) mergeInto(base config.LLMConfig) config.LLMConfig {
	out := base
	if p := strings.TrimSpace(g.Provider); p != "" {
		out.Provider = p
	}
	if m := strings.TrimSpace(g.Model); m != "" {
		out.Model = m
	}
	if id := strings.TrimSpace(g.CatalogModelID); id != "" {
		out.CatalogModelID = id
	} else if g.UseOpsModel != nil && *g.UseOpsModel {
		out.CatalogModelID = ""
	}
	if g.UseOpsModel != nil {
		out.UseOpsModel = g.UseOpsModel
	}
	return out
}

func (u *Settings) mergeInto(base config.LLMConfig) config.LLMConfig {
	if u == nil {
		return base
	}
	out := base
	if p := strings.TrimSpace(u.Provider); p != "" {
		out.Provider = p
	}
	if m := strings.TrimSpace(u.Model); m != "" {
		out.Model = m
	}
	if id := strings.TrimSpace(u.CatalogModelID); id != "" {
		out.CatalogModelID = id
	} else if u.UseOpsModel != nil && *u.UseOpsModel {
		out.CatalogModelID = ""
	}
	if u.UseOpsModel != nil {
		out.UseOpsModel = u.UseOpsModel
	}
	switch strings.ToLower(strings.TrimSpace(u.Thinking)) {
	case "on":
		v := true
		out.Thinking = &v
	case "off":
		v := false
		out.Thinking = &v
	case "auto":
		out.Thinking = nil
	}
	if u.Temperature != nil && *u.Temperature > 0 {
		out.Temperature = *u.Temperature
	}
	if u.MaxTokens != nil && *u.MaxTokens > 0 {
		out.MaxTokens = *u.MaxTokens
	}
	return out
}

func (u *Settings) touchUpdatedAt() {
	if u != nil {
		u.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
}
