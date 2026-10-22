package app

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/admin"
)

func TestPickCatalogComposeModelSkipsM3AndReasoner(t *testing.T) {
	models := []admin.ConfiguredModel{
		{Name: "MiniMax-M3", Token: "sk", DisplayName: "M3"},
		{Name: "deepseek-reasoner", Token: "sk"},
		{Name: "deepseek-chat", Token: "sk", DisplayName: "V3 Chat"},
	}
	got := pickCatalogComposeModel(models, signalDiagnoseComposeCatalogPrefs)
	if got.Name != "deepseek-chat" {
		t.Fatalf("picked %q", got.Name)
	}
}
