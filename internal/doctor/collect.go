package doctor

import (
	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// CollectFromConfig runs structured diagnostics for HTTP / JSON consumers.
// Returns all check rows and whether the overall result is healthy (no FAIL).
func CollectFromConfig(cfg *config.AppConfig, opts Options) ([]CheckResult, bool) {
	if cfg == nil {
		return []CheckResult{{Name: "config", OK: false, Detail: "app config not loaded"}}, false
	}
	results := []CheckResult{{Name: "config", OK: true, Detail: "loaded"}}
	if row, ok := profileCheck(cfg); ok {
		results = append(results, row)
	}
	results = append(results, checkSecrets(cfg)...)
	if !opts.SkipConnectivity {
		results = append(results, checkConnectivity(cfg)...)
		results = append(results, checkToolProbes(cfg)...)
	}
	return results, !anyFailed(results)
}
