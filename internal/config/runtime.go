package config

import (
	"os"
	"strconv"
	"strings"
)

// RuntimeConfig holds agent-runtime HTTP settings.
type RuntimeConfig struct {
	Port          int
	APIKey        string
	AllowInsecure bool
	ServiceName   string
	ConfigPath    string
	CORSOrigins   []string
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
		CORSOrigins:   parseCSVEnv(os.Getenv("GEEGOO_CORS_ORIGINS")),
	}
}

func parseCSVEnv(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}
