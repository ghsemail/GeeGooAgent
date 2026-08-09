package memory

import (
	"fmt"
	"strings"
)

func applyPreMarketFromDaily(w *PreMarketWorking, code string, data map[string]any) {
	ws, ok := w.Stocks[code]
	if !ok {
		return
	}
	items, _ := data["stock_premarket"].([]any)
	if len(items) == 0 {
		return
	}
	m, ok := items[0].(map[string]any)
	if !ok {
		return
	}
	ws.PreMarketResult = str(m, "result")
	ws.PreMarketConfidence = str(m, "confidence")
	ws.PreMarketReason = str(m, "reason")
	ws.PreMarketSuggestion = str(m, "suggestion")
	ws.PreMarketReportID = str(m, "report_id")
	w.Stocks[code] = ws
}

func positionHasData(data map[string]any) bool {
	for _, k := range []string{"position", "qty", "can_sell_qty"} {
		if v, ok := data[k]; ok && v != nil && fmt.Sprint(v) != "" && fmt.Sprint(v) != "0" {
			return true
		}
	}
	if items, ok := data["items"].([]any); ok && len(items) > 0 {
		return true
	}
	return false
}

func formatPositionSummary(data map[string]any) string {
	if !positionHasData(data) {
		return "无持仓"
	}
	parts := []string{}
	for _, k := range []string{"position", "qty", "cost_price", "can_sell_qty", "pl_val", "pl_ratio"} {
		if v, ok := data[k]; ok && fmt.Sprint(v) != "" {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%v", data)
	}
	return strings.Join(parts, ", ")
}

func tickerPriceFromData(data map[string]any) float64 {
	if items, ok := data["items"].([]any); ok && len(items) > 0 {
		if m, ok := items[0].(map[string]any); ok {
			if p, ok := m["price"].(float64); ok {
				return p
			}
		}
	}
	if p, ok := data["price"].(float64); ok {
		return p
	}
	return 0
}

func botLogSummary(data map[string]any) string {
	if len(data) == 0 {
		return ""
	}
	if info, ok := data["info"].(map[string]any); ok {
		if pos, ok := info["position"].(map[string]any); ok && len(pos) > 0 {
			return fmt.Sprintf("position=%v", pos)
		}
	}
	if log, ok := data["log"].([]any); ok && len(log) > 0 {
		return fmt.Sprintf("log_entries=%d", len(log))
	}
	raw := strings.TrimSpace(fmt.Sprintf("%v", data))
	if raw == "" || raw == "map[]" || raw == "[]" {
		return ""
	}
	return truncate(raw, 500)
}

func finalizeDerivedFields(w *PreMarketWorking, ws *StockWorkspace, code string) {
	switch w.Skill {
	case "intraday_stock":
		// Intraday result/confidence are resolved at report synthesis (LLM + fallback rules).
	case "postmarket_stock":
		if ws.SessionBias == "" {
			ws.SessionBias = sessionBiasFromPct(ws.ChangePct)
		}
		if ws.VsPreMarket == "" {
			ws.VsPreMarket = vsPreMarketLocal(ws.PreMarketResult, ws.SessionBias)
		}
	}
	w.Stocks[code] = *ws
}

func sessionBiasFromPct(pct float64) string {
	if pct > 1 {
		return "bullish"
	}
	if pct < -1 {
		return "bearish"
	}
	return "neutral"
}

func vsPreMarketLocal(preResult, sessionBias string) string {
	if preResult == "" {
		return "na"
	}
	if (preResult == "long" && sessionBias == "bullish") ||
		(preResult == "short" && sessionBias == "bearish") ||
		(preResult == "neutral" && sessionBias == "neutral") {
		return "aligned"
	}
	if (preResult == "long" && sessionBias == "bearish") || (preResult == "short" && sessionBias == "bullish") {
		return "contradicted"
	}
	return "partial"
}
