package runtimeapi

import "testing"

func TestCountHealthyEnabledNewsSourcesDedupesBySourceID(t *testing.T) {
	sources := map[string]any{
		"regions": map[string]any{
			"US": map[string]any{
				"market_sources": []any{
					map[string]any{"id": "finnhub", "enabled": true},
					map[string]any{"id": "cnbc_rss", "enabled": true},
				},
				"stock_sources": []any{
					map[string]any{"id": "finnhub", "enabled": true},
					map[string]any{"id": "yahoo_rss", "enabled": true},
				},
			},
		},
	}
	health := map[string]any{
		"sources": []any{
			map[string]any{"id": "finnhub", "ok": true},
			map[string]any{"id": "cnbc_rss", "ok": true},
			map[string]any{"id": "yahoo_rss", "ok": true},
		},
	}
	enabled := countEnabledNewsSources(sources)
	healthy := countHealthyEnabledNewsSources(sources, health)
	if enabled != 3 {
		t.Fatalf("enabled=%d want 3 (finnhub deduped)", enabled)
	}
	if healthy != 3 {
		t.Fatalf("healthy=%d want 3", healthy)
	}
}

func TestCountEnabledNewsSourcesUnifiedCatalog(t *testing.T) {
	sources := map[string]any{
		"catalog": []any{
			map[string]any{"id": "websearch", "label": "WebSearch", "enabled": true, "shared": true, "pools": []any{"cn_a", "crypto"}},
			map[string]any{"id": "x_public", "label": "X", "enabled": true, "shared": true},
			map[string]any{"id": "eastmoney_sector", "label": "东财板块", "enabled": true, "pools": []any{"cn_a"}},
			map[string]any{"id": "finnhub", "label": "Finnhub", "enabled": false},
		},
		"regions": map[string]any{
			"CN": map[string]any{
				"market_sources": []any{map[string]any{"id": "ignored_when_catalog", "enabled": true}},
			},
		},
	}
	if got := countEnabledNewsSources(sources); got != 3 {
		t.Fatalf("enabled=%d want 3 from catalog", got)
	}
}

func TestFleetNewsSummaryDedupesSharedAcrossNodes(t *testing.T) {
	fleetEnabled := make(map[string]bool)
	fleetHealthy := make(map[string]bool)
	shared := map[string]any{
		"catalog": []any{
			map[string]any{"id": "websearch", "enabled": true, "shared": true},
			map[string]any{"id": "x_public", "enabled": true, "shared": true},
		},
	}
	health := map[string]any{
		"sources": []any{
			map[string]any{"id": "websearch", "ok": true},
			map[string]any{"id": "x_public", "ok": false, "error": "missing API key"},
		},
	}
	cnOnly := map[string]any{
		"catalog": []any{
			map[string]any{"id": "websearch", "enabled": true},
			map[string]any{"id": "eastmoney_sector", "enabled": true},
		},
	}
	recordFleetNewsSourceIDs(fleetEnabled, fleetHealthy, shared, health)
	recordFleetNewsSourceIDs(fleetEnabled, fleetHealthy, shared, health)
	recordFleetNewsSourceIDs(fleetEnabled, fleetHealthy, cnOnly, health)
	total, healthy := fleetNewsSummaryCounts(fleetEnabled, fleetHealthy)
	if total != 3 {
		t.Fatalf("fleet total=%d want 3 (websearch,x_public,eastmoney_sector)", total)
	}
	if healthy != 1 {
		t.Fatalf("fleet healthy=%d want 1 (websearch only)", healthy)
	}
}

func TestCountHealthyEnabledNewsSourcesPartialFailure(t *testing.T) {
	sources := map[string]any{
		"regions": map[string]any{
			"CN": map[string]any{
				"market_sources": []any{map[string]any{"id": "sina_roll", "enabled": true}},
				"stock_sources":  []any{map[string]any{"id": "sina_roll", "enabled": true}},
			},
		},
	}
	health := map[string]any{
		"sources": []any{map[string]any{"id": "sina_roll", "ok": false, "error": "http 403"}},
	}
	if got := countEnabledNewsSources(sources); got != 1 {
		t.Fatalf("enabled=%d want 1", got)
	}
	if got := countHealthyEnabledNewsSources(sources, health); got != 0 {
		t.Fatalf("healthy=%d want 0", got)
	}
}
