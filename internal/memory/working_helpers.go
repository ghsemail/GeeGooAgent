package memory

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/stockfmt"
)

func applyPreMarketFromDaily(w *PreMarketWorking, code string, data map[string]any) {
	ws, ok := w.Stocks[code]
	if !ok {
		return
	}
	m, ok := firstMapFromSlice(data["stock_premarket"])
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

// firstMapFromSlice accepts both []map[string]any (tool Data from typed MCP structs)
// and []any (JSON-decoded payloads).
func firstMapFromSlice(v any) (map[string]any, bool) {
	switch items := v.(type) {
	case []map[string]any:
		if len(items) == 0 {
			return nil, false
		}
		return items[0], items[0] != nil
	case []any:
		if len(items) == 0 {
			return nil, false
		}
		m, ok := items[0].(map[string]any)
		return m, ok
	default:
		return nil, false
	}
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
	qty := stockfmt.FloatFromAny(data["position"])
	if qty == 0 {
		qty = stockfmt.FloatFromAny(data["qty"])
	}
	cost := stockfmt.FloatFromAny(data["cost_price"])
	canSell := stockfmt.FloatFromAny(data["can_sell_qty"])
	plVal := stockfmt.FloatFromAny(data["pl_val"])
	plRatio := stockfmt.FloatFromAny(data["pl_ratio"])

	parts := make([]string, 0, 4)
	if qty > 0 {
		parts = append(parts, fmt.Sprintf("持仓 %s 股", stockfmt.FormatQty(qty)))
	}
	if cost > 0 {
		parts = append(parts, fmt.Sprintf("成本价 %s", stockfmt.FormatPrice(cost)))
	}
	if canSell > 0 && (qty == 0 || canSell != qty) {
		parts = append(parts, fmt.Sprintf("可卖 %s 股", stockfmt.FormatQty(canSell)))
	}
	if plText := formatPositionPL(plVal, plRatio); plText != "" {
		parts = append(parts, plText)
	}
	if len(parts) == 0 {
		return "有持仓，明细暂缺"
	}
	return strings.Join(parts, "，")
}

func formatPositionPL(plVal, plRatio float64) string {
	if plVal == 0 && plRatio == 0 {
		return ""
	}
	if plVal != 0 && plRatio != 0 {
		return fmt.Sprintf("浮动盈亏 %s（%s）", stockfmt.FormatSignedMoneyCN(plVal), stockfmt.FormatPercent(plRatio))
	}
	if plVal != 0 {
		return fmt.Sprintf("浮动盈亏 %s", stockfmt.FormatSignedMoneyCN(plVal))
	}
	return fmt.Sprintf("浮动盈亏比例 %s", stockfmt.FormatPercent(plRatio))
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
