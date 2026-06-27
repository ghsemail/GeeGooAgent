package config

import (
	"os"
	"strconv"
)

// RuntimeConfig holds agent-runtime HTTP settings.
type RuntimeConfig struct {
	Port            int
	APIKey          string
	AllowInsecure   bool
	ServiceName     string
	ConfigPath      string
}

// LoadRuntime reads agent-runtime env (GeeGoo 34xx, not Trading legacy).
func LoadRuntime() RuntimeConfig {
	port := 3400
	if v := os.Getenv("GEEGOO_AGENT_RUNTIME_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			port = p
		}
	}
	insecure := os.Getenv("GEEGOO_AGENT_ALLOW_INSECURE_AUTH") == "true"
	return RuntimeConfig{
		Port:          port,
		APIKey:        os.Getenv("GEEGOO_AGENT_RUNTIME_API_KEY"),
		AllowInsecure: insecure,
		ServiceName:   "agent-runtime",
		ConfigPath:    DefaultPath(),
	}
}
