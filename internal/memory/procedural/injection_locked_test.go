package procedural

import "testing"

func TestCatalogEntriesForLockedSkill(t *testing.T) {
	all := []CatalogEntry{
		{ID: "signal_diagnose", Description: "a"},
		{ID: "strategy_dev", Description: "b"},
	}
	narrow := CatalogEntriesForLockedSkill(all, "signal_diagnose")
	if len(narrow) != 1 || narrow[0].ID != "signal_diagnose" {
		t.Fatalf("narrow: %+v", narrow)
	}
	if len(CatalogEntriesForLockedSkill(all, "")) != 2 {
		t.Fatal("empty lock should keep full catalog")
	}
}
