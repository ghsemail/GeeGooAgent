package mcp

import "testing"

func TestParseSearchCodeItemNestedName(t *testing.T) {
	t.Parallel()
	item := parseSearchCodeItem(map[string]any{
		"code":       "SPACEX.US",
		"market":     "US",
		"lot_size":   float64(1),
		"stock_type": "STOCK",
		"name": map[string]any{
			"en":    "SPACEX",
			"init":  "太空探索",
			"zh_hk": "太空探索",
		},
	})
	if item.Code != "SPACEX.US" {
		t.Fatalf("code: %s", item.Code)
	}
	if item.Name != "太空探索" {
		t.Fatalf("display name: %s", item.Name)
	}
	if item.NameEN != "SPACEX" {
		t.Fatalf("en: %s", item.NameEN)
	}
	if item.Market != "US" {
		t.Fatalf("market: %s", item.Market)
	}
}
