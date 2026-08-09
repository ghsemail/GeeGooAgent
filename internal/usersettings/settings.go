// Package usersettings loads per-user LLM prefs (file store) for Chat and IM Gateway.
package usersettings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

var safeUserID = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type GatewayEntry struct {
	Provider       string `json:"provider,omitempty"`
	Model          string `json:"model,omitempty"`
	CatalogModelID string `json:"catalog_model_id,omitempty"`
	UseOpsModel    *bool  `json:"use_ops_model,omitempty"`
}

type Doc struct {
	Provider       string                  `json:"provider,omitempty"`
	Model          string                  `json:"model,omitempty"`
	CatalogModelID string                  `json:"catalog_model_id,omitempty"`
	UseOpsModel    *bool                   `json:"use_ops_model,omitempty"`
	Gateways       map[string]GatewayEntry `json:"gateways,omitempty"`
}

func Path(outputDir, userID string) string {
	if strings.TrimSpace(outputDir) == "" {
		outputDir = "."
	}
	safe := safeUserID.ReplaceAllString(strings.TrimSpace(userID), "_")
	if safe == "" {
		safe = "anonymous"
	}
	return filepath.Join(outputDir, "user_llm_settings", safe+".json")
}

func Load(outputDir, userID string) (*Doc, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(Path(outputDir, userID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var doc Doc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

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

func Apply(base config.LLMConfig, doc *Doc, gateway string) config.LLMConfig {
	if doc == nil {
		return base
	}
	out := mergeTop(base, doc)
	gwKey := NormalizeGateway(gateway)
	if gwKey != "" && doc.Gateways != nil {
		if gw, ok := doc.Gateways[gwKey]; ok {
			out = mergeGW(out, gw)
		}
	}
	return out
}

func mergeTop(base config.LLMConfig, u *Doc) config.LLMConfig {
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
	return out
}

func mergeGW(base config.LLMConfig, g GatewayEntry) config.LLMConfig {
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
