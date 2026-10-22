package app

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/clients/admin"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// Benchmark-backed compose default order (server 2026-10): flash → M3 disabled → M2.7.
var signalDiagnoseComposeCatalogPrefs = []string{
	"deepseek-v4-flash",
	"MiniMax-M3",
	"MiniMax-M2.7",
}

func catalogModelNames(m admin.ConfiguredModel) (name, display string) {
	name = strings.TrimSpace(m.Name)
	display = strings.TrimSpace(m.DisplayName)
	return name, display
}

func catalogNameEquals(m admin.ConfiguredModel, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	name, display := catalogModelNames(m)
	if strings.EqualFold(name, want) {
		return true
	}
	if display != "" && strings.EqualFold(display, want) {
		return true
	}
	return false
}

func isComposeCatalogExcluded(m admin.ConfiguredModel) bool {
	name, display := catalogModelNames(m)
	blob := strings.ToLower(name + " " + display + " " + m.Kind + " " + m.Type)
	if strings.TrimSpace(m.Token) == "" {
		return true
	}
	if strings.Contains(blob, "embedding") {
		return true
	}
	kind := strings.ToLower(strings.TrimSpace(m.Kind))
	if kind == "decision" || strings.Contains(kind, "jev") {
		return true
	}
	typ := strings.ToLower(strings.TrimSpace(m.Type))
	if typ != "" && typ != "chat" && typ != "llm" && typ != "text" {
		return true
	}
	lowerName := strings.ToLower(name)
	if strings.Contains(lowerName, "-pro") || strings.Contains(lowerName, "reasoner") {
		return true
	}
	if llm.ModelCatalogLikelyForcedReasoning(name+" "+display) {
		return true
	}
	return false
}

func pickCatalogComposeModel(models []admin.ConfiguredModel, prefs []string) admin.ConfiguredModel {
	for _, pref := range prefs {
		for _, m := range models {
			if isComposeCatalogExcluded(m) {
				continue
			}
			if catalogNameEquals(m, pref) {
				return m
			}
		}
	}
	return admin.ConfiguredModel{}
}
