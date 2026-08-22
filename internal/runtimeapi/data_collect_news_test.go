package runtimeapi

import "testing"

func TestCountHealthyEnabledNewsSourcesMatchesSlots(t *testing.T) {
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
	if enabled != 4 {
		t.Fatalf("enabled=%d want 4", enabled)
	}
	if healthy != 4 {
		t.Fatalf("healthy=%d want 4", healthy)
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
	if got := countHealthyEnabledNewsSources(sources, health); got != 0 {
		t.Fatalf("healthy=%d want 0", got)
	}
}
