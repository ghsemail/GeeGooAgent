package tools

import "testing"

func TestBuildCatalogGroupsBotTools(t *testing.T) {
	r := NewRegistry()
	RegisterAll(r, Deps{})
	items := BuildCatalog(r, ChatToolNames)
	var found *CatalogItem
	for i := range items {
		if items[i].Name == "list_smart_trades" {
			found = &items[i]
			break
		}
	}
	if found == nil {
		t.Fatal("list_smart_trades missing from catalog")
	}
	if found.Domain != string(DomainBotManager) {
		t.Fatalf("domain=%s", found.Domain)
	}
	if !found.RequiresMCP {
		t.Fatal("expected requires_mcp")
	}
	if found.HTTPPath != "/getAllSmartTrades" {
		t.Fatalf("path=%q", found.HTTPPath)
	}
	if len(found.ContextInjections) == 0 {
		t.Fatal("expected context injections")
	}
}

func TestBuildToolsetSummaries(t *testing.T) {
	summaries := BuildToolsetSummaries()
	if len(summaries) < 5 {
		t.Fatalf("want toolsets, got %d", len(summaries))
	}
}
