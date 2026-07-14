package llm

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/admin"
)

func TestPickCatalogModel(t *testing.T) {
	models := []admin.ConfiguredModel{
		{ModelID: "a", Name: "gpt-5.5", Type: "configured"},
		{ModelID: "b", Name: "deepseek-v4", Type: "active"},
	}
	id, err := PickCatalogModel("2", models, "")
	if err != nil || id != "b" {
		t.Fatalf("pick by index: id=%q err=%v", id, err)
	}
	id, err = PickCatalogModel("default", models, "b")
	if err != nil || id != "" {
		t.Fatalf("pick default: id=%q err=%v", id, err)
	}
	if _, err = PickCatalogModel("9", models, ""); err == nil {
		t.Fatal("expected invalid index error")
	}
}

func TestActiveCatalogModelID(t *testing.T) {
	models := []admin.ConfiguredModel{{ModelID: "a", Type: "configured"}}
	if got := ActiveCatalogModelID("x", models); got != "x" {
		t.Fatalf("explicit: %q", got)
	}
	if got := ActiveCatalogModelID("", models); got != "a" {
		t.Fatalf("configured: %q", got)
	}
}
