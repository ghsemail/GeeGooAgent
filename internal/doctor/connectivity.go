package doctor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func checkConnectivity(cfg *config.AppConfig) []CheckResult {
	var results []CheckResult
	for _, w := range cfg.LegacyPortWarnings() {
		results = append(results, CheckResult{
			Name:   "legacy port",
			OK:     false,
			Detail: w,
		})
	}
	results = append(results, checkHTTPGet("GeeGooBot mcp-api /health", cfg.EffectiveMCPURL()+"/health", "", 15))
	results = append(results, checkHTTPGet("GeeGooBot mcp-api /ready", cfg.EffectiveMCPURL()+"/ready", "", 15))
	results = append(results, checkMCPPost(cfg, "checkTradingDay", map[string]any{
		"mcp_token": cfg.MCPToken(),
		"code":      "00700.HK",
	}))
	results = append(results, checkHTTPGet("GeeGooSignal signal /health", cfg.SignalAPIURL()+"/health", cfg.SignalAPIKey(), 15))
	results = append(results, checkHTTPGet("GeeGooSignal catalog /health", cfg.SignalCatalogURL()+"/health", "", 15))
	results = append(results, checkHTTPGet("GeeGooSignal analyze /health", cfg.SignalAnalyzeURL()+"/health", "", 15))
	results = append(results, checkHTTPGet("GeeGooData /health", cfg.DataHTTPURL()+"/health", "", 15))

	runtimeURL := strings.TrimSuffix(runtimeHealthURL(), "/")
	results = append(results, checkHTTPGet("agent-runtime /health", runtimeURL+"/health", "", 10))
	results = append(results, checkHTTPGet("agent-runtime /ready", runtimeURL+"/ready", "", 10))
	return results
}

func runtimeHealthURL() string {
	port := "3400"
	return fmt.Sprintf("http://127.0.0.1:%s", port)
}

func checkHTTPGet(name, url, bearer string, timeoutSec int) CheckResult {
	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return CheckResult{Name: name, OK: false, Detail: err.Error()}
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := client.Do(req)
	if err != nil {
		return CheckResult{Name: name, OK: false, Detail: err.Error()}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
	ok := resp.StatusCode >= 200 && resp.StatusCode < 300
	detail := fmt.Sprintf("HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	return CheckResult{Name: name, OK: ok, Detail: detail}
}

func checkMCPPost(cfg *config.AppConfig, route string, body map[string]any) CheckResult {
	name := "GeeGooBot mcp " + route
	url := cfg.EffectiveMCPURL() + "/" + route
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return CheckResult{Name: name, OK: false, Detail: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.MCPAPIKey())
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return CheckResult{Name: name, OK: false, Detail: err.Error()}
	}
	defer resp.Body.Close()
	preview, _ := io.ReadAll(io.LimitReader(resp.Body, 120))
	ok := resp.StatusCode == http.StatusOK
	detail := fmt.Sprintf("HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(preview)))
	return CheckResult{Name: name, OK: ok, Detail: detail}
}
