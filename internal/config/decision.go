package config

import "strings"

// DecisionConfig holds TypeSafe Jev (System One) credentials from ops catalog.
type DecisionConfig struct {
	TokenKey string `json:"token_key"`
	Model    string `json:"model"`
	BaseURL  string `json:"base_url,omitempty"`
}

const (
	DefaultDecisionModel  = "jev-1.13.0"
	DefaultDecisionBaseURL  = "https://api.typesafe.ai/v1"
	DefaultDecisionProvider = "typesafe"
)

// ResolvedDecision is EffectiveDecision output.
type ResolvedDecision struct {
	Provider   string
	TokenKey   string
	Model      string
	BaseURL    string
	Configured bool
}

// ResolvedDecision returns decision settings from config.
func (c *AppConfig) ResolvedDecision() ResolvedDecision {
	var cfg DecisionConfig
	if c != nil {
		cfg = c.Decision
	}
	key := strings.TrimSpace(cfg.TokenKey)
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = DefaultDecisionModel
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		base = DefaultDecisionBaseURL
	}
	return ResolvedDecision{
		Provider:   DefaultDecisionProvider,
		TokenKey:   key,
		Model:      model,
		BaseURL:    base,
		Configured: key != "" && strings.TrimSpace(cfg.Model) != "",
	}
}
