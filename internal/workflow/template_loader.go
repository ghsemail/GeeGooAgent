package workflow

import (
	"os"
	"path/filepath"
)

func loadSkillTemplate(rel string) string {
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

func loadStockPremarketTemplate() string {
	return loadSkillTemplate("skills/premarket_stock/template.md")
}
