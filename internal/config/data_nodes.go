package config

import (
	"os"
	"strings"
)

// DataNodeConfig describes one GeeGooData data-api node for BFF aggregation.
type DataNodeConfig struct {
	ID        string   `json:"id"`
	Label     string   `json:"label"`
	BaseURL   string   `json:"base_url"`
	Regions   []string `json:"regions,omitempty"`
	BearerEnv string   `json:"bearer_env,omitempty"`
}

// ResolvedDataNode is a runtime-ready data node with bearer token resolved.
type ResolvedDataNode struct {
	ID      string
	Label   string
	BaseURL string
	Regions []string
	Bearer  string
}

// ResolvedDataNodes returns configured data nodes, or a single default from data_base_url.
func (c *AppConfig) ResolvedDataNodes() []ResolvedDataNode {
	if c == nil {
		return []ResolvedDataNode{{ID: "default", Label: "GeeGooData", BaseURL: DefaultDataHTTPURL}}
	}
	if len(c.DataNodes) > 0 {
		out := make([]ResolvedDataNode, 0, len(c.DataNodes))
		for _, n := range c.DataNodes {
			base := trimSlash(strings.TrimSpace(n.BaseURL))
			if base == "" {
				continue
			}
			id := strings.TrimSpace(n.ID)
			if id == "" {
				id = "node-" + strings.TrimPrefix(base, "http://")
			}
			label := strings.TrimSpace(n.Label)
			if label == "" {
				label = id
			}
			regions := make([]string, 0, len(n.Regions))
			for _, r := range n.Regions {
				if s := strings.ToUpper(strings.TrimSpace(r)); s != "" {
					regions = append(regions, s)
				}
			}
			out = append(out, ResolvedDataNode{
				ID:      id,
				Label:   label,
				BaseURL: base,
				Regions: regions,
				Bearer:  resolveDataNodeBearer(n.BearerEnv),
			})
		}
		if len(out) > 0 {
			return out
		}
	}
	base := c.DataHTTPURL()
	return []ResolvedDataNode{{
		ID:      "default",
		Label:   "GeeGooData",
		BaseURL: base,
		Bearer:  resolveDataNodeBearer(""),
	}}
}

func (c *AppConfig) DataNodeByID(id string) (ResolvedDataNode, bool) {
	id = strings.TrimSpace(id)
	for _, n := range c.ResolvedDataNodes() {
		if n.ID == id {
			return n, true
		}
	}
	return ResolvedDataNode{}, false
}

func resolveDataNodeBearer(envKey string) string {
	if k := strings.TrimSpace(envKey); k != "" {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	if v := strings.TrimSpace(os.Getenv("GEEGOO_DATA_SERVICE_TOKEN")); v != "" {
		return v
	}
	return ""
}
