package runtimeapi

import "strings"

// News sources counting for fleet overview and per-node probes.
//
// GeeGooData (feat/stock-and-market-news-sources) returns a unified catalog on
// GET /v1/news/sources. Assumed pass-through shape (adjust with Data API if renamed):
//   - catalog: []{ id, label, enabled, shared?, category?, layers?, pools? }
//   - pools: []{ id, label }  (macro pool metadata, optional)
//   - regions: legacy CN/US/HK slots (kept for collectors; ignored for counts when catalog is non-empty)
//
// Enabled/healthy counts dedupe by source_id within a node. Fleet overview summary
// dedupes the same id across nodes (shared sources such as websearch count once).

func enabledSourceIDsFromNewsSources(sources map[string]any) map[string]struct{} {
	out := make(map[string]struct{})
	if sources == nil {
		return out
	}
	if entries, ok := newsCatalogEntries(sources); ok {
		for _, id := range enabledIDsFromCatalogEntries(entries) {
			out[id] = struct{}{}
		}
		return out
	}
	for id := range enabledSourceIDsFromRegions(sources) {
		out[id] = struct{}{}
	}
	return out
}

func countEnabledNewsSources(sources map[string]any) int {
	return len(enabledSourceIDsFromNewsSources(sources))
}

func countHealthyEnabledNewsSources(sources, health map[string]any) int {
	probeOK := probeOKBySourceID(health)
	n := 0
	for id := range enabledSourceIDsFromNewsSources(sources) {
		if probeOK[id] {
			n++
		}
	}
	return n
}

func newsCatalogEntries(sources map[string]any) ([]any, bool) {
	if list, ok := sources["catalog"].([]any); ok && len(list) > 0 {
		return list, true
	}
	if _, hasRegions := sources["regions"].(map[string]any); hasRegions {
		return nil, false
	}
	if list, ok := sources["sources"].([]any); ok && len(list) > 0 && isNewsCatalogList(list) {
		return list, true
	}
	return nil, false
}

func isNewsCatalogList(list []any) bool {
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if m["ok"] != nil {
			return false
		}
		if m["label"] != nil || m["pools"] != nil || m["shared"] != nil || m["category"] != nil {
			return true
		}
	}
	return false
}

func catalogEntryProbeExempt(m map[string]any) bool {
	if m["probe_exempt"] == true {
		return true
	}
	if strings.EqualFold(stringFromAny(m["category"]), "other") {
		return true
	}
	return stringFromAny(m["id"]) == "x_public"
}

func enabledIDsFromCatalogEntries(entries []any) []string {
	ids := make([]string, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, item := range entries {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if m["enabled"] == false || catalogEntryProbeExempt(m) {
			continue
		}
		id := stringFromAny(m["id"])
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func enabledSourceIDsFromRegions(sources map[string]any) map[string]struct{} {
	out := make(map[string]struct{})
	regions, ok := sources["regions"].(map[string]any)
	if !ok {
		return out
	}
	for _, rv := range regions {
		region, ok := rv.(map[string]any)
		if !ok {
			continue
		}
		recordEnabledListIDs(region["market_sources"], out)
		recordEnabledListIDs(region["stock_sources"], out)
		recordEnabledListIDs(region["sources"], out)
	}
	return out
}

func recordEnabledListIDs(v any, out map[string]struct{}) {
	list, ok := v.([]any)
	if !ok {
		return
	}
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if m["enabled"] == false {
			continue
		}
		id := stringFromAny(m["id"])
		if id != "" {
			out[id] = struct{}{}
		}
	}
}

func recordFleetNewsSourceIDs(fleetEnabled, fleetHealthy map[string]bool, sources, health map[string]any) {
	probeOK := probeOKBySourceID(health)
	for id := range enabledSourceIDsFromNewsSources(sources) {
		fleetEnabled[id] = true
		if probeOK[id] {
			fleetHealthy[id] = true
		}
	}
}

func fleetNewsSummaryCounts(fleetEnabled, fleetHealthy map[string]bool) (total, healthy int) {
	total = len(fleetEnabled)
	for id := range fleetEnabled {
		if fleetHealthy[id] {
			healthy++
		}
	}
	return total, healthy
}
