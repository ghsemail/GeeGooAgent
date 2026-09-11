package doctor

import (
	"fmt"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// CheckToolPlatformDrift loads the app registry and reports catalog drift (M5).
func CheckToolPlatformDrift(configPath string) []CheckResult {
	application, err := app.LoadFromConfigPath(configPath, true)
	if err != nil {
		return []CheckResult{{Name: "tools drift", OK: false, Detail: err.Error()}}
	}
	defer func() { _ = application.Close() }()
	if application.Registry == nil {
		return []CheckResult{{Name: "tools drift", OK: false, Detail: "registry nil"}}
	}
	issues := tools.CheckCatalogDrift(application.Registry)
	if len(issues) == 0 {
		return []CheckResult{{Name: "tools drift", OK: true, Detail: "catalog ↔ registry aligned"}}
	}
	return []CheckResult{{
		Name: "tools drift", OK: false,
		Detail: tools.FormatDriftIssues(issues),
	}}
}

// CheckToolPolicy reports loaded declarative policy rules.
func CheckToolPolicy() CheckResult {
	count := tools.PolicyRuleCount()
	return CheckResult{
		Name:   "tool policy",
		OK:     true,
		Detail: fmt.Sprintf("%d rule(s) loaded", count),
	}
}
