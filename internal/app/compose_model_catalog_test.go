package app

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/admin"
)

func TestCatalogNameEqualsNotSubstring(t *testing.T) {
	models := []admin.ConfiguredModel{
		{Name: "deepseek-v4-pro", Token: "sk"},
		{Name: "deepseek-v4-flash", Token: "sk"},
	}
	got := pickCatalogComposeModel(models, []string{"deepseek-v4-flash"})
	if got.Name != "deepseek-v4-flash" {
		t.Fatalf("picked %q", got.Name)
	}
}

func TestComposeExcludeProEmbeddingDecision(t *testing.T) {
	models := []admin.ConfiguredModel{
		{Name: "deepseek-v4-pro", Token: "sk"},
		{Name: "kinfra-text-embedding-4b", Token: "sk"},
		{Name: "jev", Token: "sk", Kind: "decision"},
		{Name: "deepseek-v4-flash", Token: "sk", Type: "chat"},
	}
	got := pickCatalogComposeModel(models, signalDiagnoseComposeCatalogPrefs)
	if got.Name != "deepseek-v4-flash" {
		t.Fatalf("picked %q", got.Name)
	}
}
