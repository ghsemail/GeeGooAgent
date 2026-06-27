package doctor

import (
	"fmt"
	"os"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// CheckResult is one diagnostic line.
type CheckResult struct {
	Name   string
	OK     bool
	Detail string
}

// Run performs A0 doctor checks (config file presence and parse).
func Run(configPath string) int {
	results := []CheckResult{}
	cfg, cfgResults := checkConfigFile(configPath)
	results = append(results, cfgResults...)
	printResults(results)
	if cfg == nil {
		return 1
	}

	secretResults := checkSecrets(cfg)
	printResults(secretResults)
	if anyFailed(secretResults) {
		fmt.Println("\n提示: 运行 geegoo setup 填写 mcp_token、llm.token_key 与 API Bearer。")
		return 1
	}

	fmt.Println("\n配置检查通过（A0）。完整 API/LLM 检查将在后续阶段启用。")
	return 0
}

func checkConfigFile(path string) (*config.AppConfig, []CheckResult) {
	if _, err := os.Stat(path); err != nil {
		return nil, []CheckResult{{Name: "config", OK: false, Detail: fmt.Sprintf("not found: %s", path)}}
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, []CheckResult{{Name: "config", OK: false, Detail: err.Error()}}
	}
	return cfg, []CheckResult{{Name: "config", OK: true, Detail: path}}
}

func checkSecrets(cfg *config.AppConfig) []CheckResult {
	results := []CheckResult{}
	results = append(results, secretCheck("geegoo mcp api_key (sk-)", mask(cfg.MCPAPIKey()), cfg.MCPAPIKey() != "" && cfg.MCPAPIKey() != "sk-REPLACE"))
	results = append(results, secretCheck("mcp_token", cfg.MCPToken(), cfg.MCPToken() != ""))
	results = append(results, secretCheck("llm.token_key", mask(cfg.LLM.TokenKey), cfg.LLM.TokenKey != ""))
	return results
}

func secretCheck(name, detail string, ok bool) CheckResult {
	if !ok {
		detail = "missing or placeholder — run geegoo setup"
	}
	return CheckResult{Name: name, OK: ok, Detail: detail}
}

func mask(value string) string {
	if len(value) <= 6 {
		return "***"
	}
	return value[:3] + "..." + value[len(value)-3:]
}

func printResults(results []CheckResult) {
	for _, row := range results {
		mark := "OK"
		if !row.OK {
			mark = "FAIL"
		}
		fmt.Printf("  [%s] %s: %s\n", mark, row.Name, row.Detail)
	}
}

func anyFailed(results []CheckResult) bool {
	for _, r := range results {
		if !r.OK {
			return true
		}
	}
	return false
}
