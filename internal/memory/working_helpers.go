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
	items, _ := data["pre_market"].([]any)
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
	if info, ok := data["info"].(map[string]any); ok {
		if pos, ok := info["position"].(map[string]any); ok {
			return fmt.Sprintf("position=%v", pos)
		}
	}
	if log, ok := data["log"].([]any); ok && len(log) > 0 {
		return fmt.Sprintf("log_entries=%d", len(log))
	}
	return truncate(fmt.Sprintf("%v", data), 500)
}

func finalizeDerivedFields(w *PreMarketWorking, ws *StockWorkspace, code string) {
	switch w.Skill {
	case "intraday":
		if ws.IntradayResult == "" {
			ws.IntradayResult, ws.IntradayConfidence = decideIntradayLocal(*ws)
		}
	case "post_market":
		if ws.SessionBias == "" {
			ws.SessionBias = sessionBiasFromPct(ws.ChangePct)
		}
		if ws.VsPreMarket == "" {
			ws.VsPreMarket = vsPreMarketLocal(ws.PreMarketResult, ws.SessionBias)
		}
	}
	w.Stocks[code] = *ws
}

func decideIntradayLocal(ws StockWorkspace) (string, string) {
	result, confidence := "hold", "medium"
	isBuy := strings.Contains(ws.TradeType, "买") || strings.Contains(strings.ToLower(ws.TradeType), "buy")
	isSell := strings.Contains(ws.TradeType, "卖") || strings.Contains(strings.ToLower(ws.TradeType), "sell")
	reminder := strings.Contains(strings.ToLower(ws.BotType), "reminder")
	if isSell && !reminder && !ws.HasPosition {
		return result, "low"
	}
	if isBuy {
		if ws.PreMarketResult == "short" && ws.PreMarketConfidence == "high" {
			return result, "low"
		}
		if ws.PreMarketResult == "" || ws.PreMarketResult == "long" || ws.PreMarketResult == "neutral" {
			return "buy", confidence
		}
	}
	if isSell && (reminder || ws.PreMarketResult == "short" || ws.PreMarketResult == "neutral" || ws.PreMarketResult == "") {
		return "sell", confidence
	}
	return result, confidence
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
