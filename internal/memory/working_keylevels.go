package memory

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/stockfmt"
)

func applyKeyLevelsTool(ws *StockWorkspace, data map[string]any, ok bool) {
	if ws == nil || data == nil {
		return
	}
	ws.KeyLevelsEngineOK = ok
	if cp := stockfmt.FloatFromAny(data["current_price"]); cp > 0 {
		ws.CurrentPrice = cp
		ws.PriceSource = "key_level_engine"
	} else if cp := stockfmt.FloatFromAny(data["price"]); cp > 0 {
		ws.CurrentPrice = cp
		ws.PriceSource = "key_level_engine"
	}
	legacy := legacyMap(data)
	if s := stockfmt.FloatFromAny(legacy["QFLSupport"]); s > 0 {
		ws.KeyLevelSupportCenter = s
	}
	if r := stockfmt.FloatFromAny(legacy["QFLResistance"]); r > 0 {
		ws.KeyLevelResistanceCenter = r
	}
	if judgment, _ := data["judgment"].(map[string]any); judgment != nil {
		applyJudgmentSide(ws, judgment["support"], true)
		applyJudgmentSide(ws, judgment["resistance"], false)
	}
	levels, _ := data["levels"].(map[string]any)
	cands, _ := data["candidates"].(map[string]any)
	if levels != nil {
		applyFirstZone(ws, levels["support"], true)
		applyFirstZone(ws, levels["resistance"], false)
	} else if cands != nil {
		applyFirstZone(ws, cands["support"], true)
		applyFirstZone(ws, cands["resistance"], false)
	} else {
		kl := keyLevelsMap(data)
		if cp := stockfmt.FloatFromAny(kl["current_price"]); cp > 0 && ws.CurrentPrice <= 0 {
			ws.CurrentPrice = cp
		}
		zones, _ := kl["zones"].(map[string]any)
		if zones != nil {
			applyFirstZone(ws, zones["supports"], true)
			applyFirstZone(ws, zones["resistances"], false)
		}
	}
}

func applyJudgmentSide(ws *StockWorkspace, raw any, support bool) {
	m, ok := raw.(map[string]any)
	if !ok {
		return
	}
	center := stockfmt.FloatFromAny(m["center"])
	low := stockfmt.FloatFromAny(m["low"])
	high := stockfmt.FloatFromAny(m["high"])
	state, _ := m["state"].(string)
	sources := stringSliceField(m["sources"])
	zone := ""
	if low > 0 && high > 0 {
		zone = stockfmt.FormatPriceRange(low, high)
	}
	if support {
		if center > 0 {
			ws.KeyLevelSupportCenter = center
		}
		if zone != "" {
			ws.KeyLevelSupportZone = zone
		}
		ws.KeyLevelSupportState = state
		ws.KeyLevelSupportSources = strings.Join(sources, ",")
	} else {
		if center > 0 {
			ws.KeyLevelResistanceCenter = center
		}
		if zone != "" {
			ws.KeyLevelResistZone = zone
		}
		ws.KeyLevelResistState = state
		ws.KeyLevelResistSources = strings.Join(sources, ",")
	}
}

func keyLevelsMap(data map[string]any) map[string]any {
	if kl, ok := data["key_levels"].(map[string]any); ok && kl != nil {
		return kl
	}
	return data
}

func legacyMap(data map[string]any) map[string]any {
	if leg, ok := data["legacy"].(map[string]any); ok && leg != nil {
		return leg
	}
	return data
}

func applyFirstZone(ws *StockWorkspace, raw any, support bool) {
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return
	}
	m, ok := items[0].(map[string]any)
	if !ok {
		return
	}
	low := stockfmt.FloatFromAny(m["low"])
	high := stockfmt.FloatFromAny(m["high"])
	center := stockfmt.FloatFromAny(m["center"])
	state, _ := m["state"].(string)
	sources := stringSliceField(m["sources"])
	zone := ""
	if low > 0 && high > 0 {
		zone = stockfmt.FormatPriceRange(low, high)
	}
	if support {
		if center > 0 {
			ws.KeyLevelSupportCenter = center
		}
		if zone != "" {
			ws.KeyLevelSupportZone = zone
		}
		ws.KeyLevelSupportState = state
		ws.KeyLevelSupportSources = strings.Join(sources, ",")
	} else {
		if center > 0 {
			ws.KeyLevelResistanceCenter = center
		}
		if zone != "" {
			ws.KeyLevelResistZone = zone
		}
		ws.KeyLevelResistState = state
		ws.KeyLevelResistSources = strings.Join(sources, ",")
	}
}

func formatKeyLevelEvidenceSummary(ws StockWorkspace) string {
	if ws.KeyLevelSupportZone != "" && ws.KeyLevelResistZone != "" {
		return "支撑带 " + ws.KeyLevelSupportZone + "；阻力带 " + ws.KeyLevelResistZone
	}
	if ws.KeyLevelSupportCenter > 0 && ws.KeyLevelResistanceCenter > 0 {
		return stockfmt.FormatPriceRange(ws.KeyLevelSupportCenter, ws.KeyLevelResistanceCenter)
	}
	return "key levels"
}

func stringSliceField(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}
