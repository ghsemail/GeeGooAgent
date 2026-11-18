package app

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/admin"
)

func TestCatalogNameEqualsNotSubstring(t *testing.T) {
	models := []admin.ConfiguredModel{
		{Name: "deepseek-v4-pro", Token: "sk"},
		{Name: "deepseek-flash", Token: "sk"},
	}
	got := pickCatalogComposeModel(models, []string{"deepseek-flash"})
	if got.Name != "deepseek-flash" {
		t.Fatalf("picked %q", got.Name)
	}
}

func TestComposeExcludeProEmbeddingDecision(t *testing.T) {
	models := []admin.ConfiguredModel{
		{Name: "deepseek-v4-pro", Token: "sk"},
		{Name: "kinfra-text-embedding-4b", Token: "sk"},
		{Name: "jev", Token: "sk", Kind: "decision"},
		{Name: "deepseek-flash", Token: "sk", Type: "chat"},
	}
	got := pickCatalogComposeModel(models, signalDiagnoseComposeCatalogPrefs)
	if got.Name != "deepseek-flash" {
		t.Fatalf("picked %q", got.Name)
	}
}
