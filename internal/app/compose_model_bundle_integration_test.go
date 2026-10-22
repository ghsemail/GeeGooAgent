package app

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/admin"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func TestResolveComposeBundleCatalogPref(t *testing.T) {
	a := &App{Config: &config.AppConfig{LLM: config.LLMConfig{Provider: "openai", TokenKey: "sk", Model: "chat"}}}
	a.Agent = nil
	// Simulate catalog pick path without HTTP: inject via pick function test already covers flash.
	models := []admin.ConfiguredModel{
		{Name: "deepseek-v4-pro", Token: "x", Type: "remote"},
		{Name: "deepseek-v4-flash", Token: "y", Type: "remote"},
	}
	doc, _ := pickCatalogComposeModelWithDetail(models, signalDiagnoseComposeCatalogPrefs)
	if doc.Name != "deepseek-v4-flash" {
		t.Fatalf("pick %q", doc.Name)
	}
	p, err := a.composeProviderFromCatalogDoc(doc)
	if err != nil {
		t.Fatal(err)
	}
	if p.Model() != "deepseek-v4-flash" {
		t.Fatalf("model %q", p.Model())
	}
	_ = llm.ClassifyProviderFromGateway
}
