package memory

import "strings"

// ApplyPremarketNotifySnapshot stores fields used by Feishu stock digest.
func ApplyPremarketNotifySnapshot(w *PreMarketWorking, code string, args map[string]any) {
	if w == nil || code == "" {
		return
	}
	ws, ok := w.Stocks[code]
	if !ok {
		return
	}
	ws.PreMarketResult = strMap(args, "result", ws.PreMarketResult)
	ws.PreMarketConfidence = strMap(args, "confidence", ws.PreMarketConfidence)
	ws.PreMarketReason = strMap(args, "reason", ws.PreMarketReason)
	ws.PreMarketSuggestion = strMap(args, "suggestion", ws.PreMarketSuggestion)
	ws.ReportSummary = strMap(args, "summary", ws.ReportSummary)
	w.Stocks[code] = ws
}

// ApplyPostmarketNotifySnapshot stores postmarket digest fields for Feishu.
func ApplyPostmarketNotifySnapshot(w *PreMarketWorking, code string, args map[string]any) {
	if w == nil || code == "" {
		return
	}
	ws, ok := w.Stocks[code]
	if !ok {
		return
	}
	ws.ReportSummary = strMap(args, "summary", ws.ReportSummary)
	ws.ReportMarketSummary = strMap(args, "market_summary", ws.ReportMarketSummary)
	ws.ReportTradeSummary = strMap(args, "trade_summary", ws.ReportTradeSummary)
	ws.ReportExperienceSummary = strMap(args, "experience_summary", ws.ReportExperienceSummary)
	w.Stocks[code] = ws
}

func strMap(m map[string]any, key, fallback string) string {
	if m == nil {
		return strings.TrimSpace(fallback)
	}
	if v, ok := m[key].(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(fallback)
}
