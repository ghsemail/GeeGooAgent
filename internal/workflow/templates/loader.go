package templates

import (
	"os"
	"path/filepath"
)

// LoadSkillTemplate reads a skill template relative to the repo root.
func LoadSkillTemplate(rel string) string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := wd
	for i := 0; i < 8; i++ {
		path := filepath.Join(dir, rel)
		if raw, err := os.ReadFile(path); err == nil {
			return string(raw)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// LoadStockPremarketTemplate loads skills/premarket_stock/template.md.
func LoadStockPremarketTemplate() string {
	return LoadSkillTemplate("skills/premarket_stock/template.md")
}

// LoadMarketReportTemplate loads skills/premarket_market/template.md.
func LoadMarketReportTemplate() string {
	return LoadSkillTemplate("skills/premarket_market/template.md")
}

// LoadStrategyArchiveTemplate loads skills/generate_strategy_archive/template.md.
func LoadStrategyArchiveTemplate() string {
	return LoadSkillTemplate("skills/generate_strategy_archive/template.md")
}
