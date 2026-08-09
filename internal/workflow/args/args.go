package args

import (
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
)

// DefaultTradingDayCode is the fallback symbol for trading-day checks.
const DefaultTradingDayCode = "00700.HK"

var timeNow = func() time.Time { return time.Now() }

// SetTimeNowForTest overrides the clock used by ReportDateFor.
func SetTimeNowForTest(fn func() time.Time) { timeNow = fn }

// StockCodeArg builds get_current_price / get_position args.
func StockCodeArg(w *memory.PreMarketWorking) map[string]any {
	return map[string]any{"code": w.CurrentStock}
}

// StockReportDateArg builds report_date scoped stock report list args.
func StockReportDateArg(w *memory.PreMarketWorking) map[string]any {
	return map[string]any{"code": w.CurrentStock, "report_date": ReportDateFor(w, w.CurrentStock)}
}

// MCPHourlyArg builds a single-slot hourly MCP analysis call.
func MCPHourlyArg(promptID, slot string) func(*memory.PreMarketWorking) map[string]any {
	return func(w *memory.PreMarketWorking) map[string]any {
		ws := w.Stocks[w.CurrentStock]
		return map[string]any{
			"name": ws.StockName, "code": w.CurrentStock,
			"prompt_id": promptID, "period": "hourly", "language": "cn",
			"analysis_slot": slot,
		}
	}
}

// MCPHourlyBundleArg builds get_hourly_analysis_bundle args.
func MCPHourlyBundleArg(w *memory.PreMarketWorking) map[string]any {
	ws := w.Stocks[w.CurrentStock]
	return map[string]any{
		"name": ws.StockName, "code": w.CurrentStock, "language": "cn",
	}
}

// StockCompleteArg builds the per-stock completion log entry.
func StockCompleteArg(w *memory.PreMarketWorking) map[string]any {
	ws := w.Stocks[w.CurrentStock]
	return map[string]any{
		"step":    fmt.Sprintf("stock_complete:%s", w.CurrentStock),
		"message": fmt.Sprintf("status=%s result=%s", ws.Status, ws.IntradayResult),
		"status":  "ok",
	}
}

// ReportDateFor returns the report date for a stock workspace entry.
func ReportDateFor(w *memory.PreMarketWorking, code string) string {
	if ws, ok := w.Stocks[code]; ok && strings.TrimSpace(ws.ReportDate) != "" {
		return ws.ReportDate
	}
	return timeNow().Format("2006-01-02")
}
