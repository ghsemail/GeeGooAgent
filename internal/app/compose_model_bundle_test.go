package app

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/admin"
)

func TestPickCatalogComposeModelFlashNotPro(t *testing.T) {
	models := []admin.ConfiguredModel{
		{Name: "deepseek-v4-pro", Token: "sk-pro", Type: "remote"},
		{Name: "deepseek-v4-flash", Token: "sk-flash", Type: "remote"},
		{Name: "MiniMax-M3", Token: "sk-m3", Type: "remote"},
	}
	doc, detail := pickCatalogComposeModelWithDetail(models, signalDiagnoseComposeCatalogPrefs)
	if doc.Name != "deepseek-v4-flash" {
		t.Fatalf("picked %q detail=%q", doc.Name, detail)
	}
}

func TestComposeCatalogExcludedByTypeEmbeddingNotRemote(t *testing.T) {
	m := admin.ConfiguredModel{Name: "deepseek-v4-flash", Token: "sk", Type: "remote"}
	if isComposeCatalogExcluded(m) {
		t.Fatal("remote chat row should not be excluded")
	}
	m2 := admin.ConfiguredModel{Name: "kinfra-text-embedding-4b", Token: "sk", Type: "embedding"}
	if !isComposeCatalogExcluded(m2) {
		t.Fatal("embedding should be excluded")
	}
}
