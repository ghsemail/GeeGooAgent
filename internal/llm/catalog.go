package llm

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/clients/admin"
)

// PickCatalogModel resolves /model selection against catalog rows.
// Empty choice keeps the current selection; "default"/"0" clears override (ops configured).
func PickCatalogModel(choice string, models []admin.ConfiguredModel, activeID string) (string, error) {
	text := strings.TrimSpace(choice)
	if text == "" {
		return strings.TrimSpace(activeID), nil
	}
	lower := strings.ToLower(text)
	if lower == "default" || lower == "ops" || text == "0" {
		return "", nil
	}
	if n, err := strconv.Atoi(text); err == nil {
		if n < 1 || n > len(models) {
			return "", fmt.Errorf("invalid model index: %d (1-%d)", n, len(models))
		}
		return strings.TrimSpace(models[n-1].ModelID), nil
	}
	for _, m := range models {
		if strings.EqualFold(text, m.ModelID) || strings.EqualFold(text, m.Name) || strings.EqualFold(text, m.DisplayName) {
			return strings.TrimSpace(m.ModelID), nil
		}
	}
	return "", fmt.Errorf("unknown catalog model: %s", text)
}

// CatalogModelLabel formats one row for /model listing.
func CatalogModelLabel(m admin.ConfiguredModel) string {
	name := strings.TrimSpace(m.DisplayName)
	if name == "" {
		name = strings.TrimSpace(m.Name)
	}
	if name == "" {
		name = m.ModelID
	}
	if m.Type == "configured" {
		return name + " [运营默认]"
	}
	return name
}

// ActiveCatalogModelID returns the effective catalog selection.
func ActiveCatalogModelID(explicitID string, models []admin.ConfiguredModel) string {
	if id := strings.TrimSpace(explicitID); id != "" {
		return id
	}
	for _, m := range models {
		if m.Type == "configured" {
			return strings.TrimSpace(m.ModelID)
		}
	}
	return ""
}
