// Command probe_vs_diagnose calls probe_bot_signal_series and diagnose_bot_signal_series
// with the same Signal args and prints hit counts for parity checks.
//
// Config (first match):
//   - GEEGOO_CONFIG=/path/to/config.json  (uses signal_api_url, signal_api_key, mcp_token)
//   - SIGNAL_API_URL + SIGNAL_API_KEY env
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func main() {
	code := envOr("PROBE_CODE", "00700.HK")
	freq := envOr("PROBE_FREQ", "60m")
	months := 3

	sigURL, sigKey, mcpToken := resolveSignalCredentials()
	if sigKey == "" {
		fmt.Fprintln(os.Stderr, "signal API key missing: set GEEGOO_CONFIG (signal_api_key) or SIGNAL_API_KEY")
		os.Exit(1)
	}
	client := mcp.NewClient(sigURL, sigKey, mcp.Options{AllowedHosts: []string{"146.56.225.252", "127.0.0.1", "localhost"}})
	reg := tools.NewRegistry()
	tools.RegisterAll(reg, tools.Deps{HTTP: tools.HTTPBackends{SignalAPI: client, SignalCatalog: client}})

	buy := []any{map[string]any{"index": "SAR", "type": "signal", "param": map[string]any{"acceleration": 0.02, "maximum": 0.2}}}
	sell := buy
	probeArgs := map[string]any{
		"code": code, "frequency": freq, "months_back": months,
		"buy_signal": buy, "sell_signal": sell,
		"include_key_levels": true, "key_levels_mode": "series", "key_levels_align": "daily",
	}
	diagArgs := map[string]any{
		"code": code, "frequency": freq, "months_back": months,
		"buy_signal": buy, "sell_signal": sell, "side": "both",
	}
	toolCtx := tools.Context{MCPToken: mcpToken}
	probe := reg.Execute(tools.CallRequest{Name: "probe_bot_signal_series", Arguments: probeArgs}, toolCtx)
	diag := reg.Execute(tools.CallRequest{Name: "diagnose_bot_signal_series", Arguments: diagArgs}, toolCtx)
	fmt.Printf("code=%s freq=%s months=%d signal_url=%s\n", code, freq, months, sigURL)
	printResult("probe", probe)
	printResult("diagnose", diag)
}

func resolveSignalCredentials() (url, key, mcpToken string) {
	url = envOr("SIGNAL_API_URL", "http://146.56.225.252:3200")
	key = strings.TrimSpace(os.Getenv("SIGNAL_API_KEY"))
	mcpToken = strings.TrimSpace(os.Getenv("MCP_TOKEN"))
	if path := strings.TrimSpace(os.Getenv("GEEGOO_CONFIG")); path != "" {
		if cfg, ok := loadConfigFile(path); ok {
			if u := strings.TrimSpace(cfg.SignalAPIURL()); u != "" {
				url = u
			}
			if k := strings.TrimSpace(cfg.SignalAPIKey()); k != "" {
				key = k
			}
			if t := strings.TrimSpace(cfg.MCPToken()); t != "" {
				mcpToken = t
			}
		}
	}
	if key == "" {
		for _, path := range []string{"config.json", "/etc/geegoo/config.json", "deploy/config.json"} {
			if cfg, ok := loadConfigFile(path); ok {
				if k := strings.TrimSpace(cfg.SignalAPIKey()); k != "" {
					key = k
					if u := strings.TrimSpace(cfg.SignalAPIURL()); u != "" {
						url = u
					}
					if t := strings.TrimSpace(cfg.MCPToken()); t != "" {
						mcpToken = t
					}
					break
				}
			}
		}
	}
	return url, key, mcpToken
}

func loadConfigFile(path string) (*config.AppConfig, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var cfg config.AppConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, false
	}
	return &cfg, true
}

func printResult(label string, res tools.Result) {
	fmt.Printf("\n== %s status=%s ==\n%s\n", label, res.Status, res.Summary)
	if res.Data == nil {
		return
	}
	brief := map[string]any{
		"bar_count":    res.Data["bar_count"],
		"bars_len":     len(asAnySlice(res.Data["bars"])),
		"range":        res.Data["range"],
		"buy_hits":     res.Data["buy_hits"],
		"sell_hits":    res.Data["sell_hits"],
		"hits":         res.Data["hits"],
		"buy_rules_n":  len(asAnySlice(res.Data["buy_rules"])),
		"sell_rules_n": len(asAnySlice(res.Data["sell_rules"])),
	}
	raw, _ := json.MarshalIndent(brief, "", "  ")
	fmt.Println(string(raw))
}

func asAnySlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func envOr(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}
