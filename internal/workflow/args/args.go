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

// TradingDayProbeCode returns the symbol used for check_trading_day by market.
func TradingDayProbeCode(market string) string {
	switch strings.ToUpper(strings.TrimSpace(market)) {
	case "CN":
		return "000001.SZ"
	case "US":
		return "AAPL.US"
	default:
		return DefaultTradingDayCode
	}
}

func shanghaiLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}

// CalendarDateForMarket is the YYYY-MM-DD used as report/session date.
// CN/HK use Asia/Shanghai (agent cron TZ). US uses America/New_York so
// postmarket at 05:00 CST still dates the session that just closed.
func CalendarDateForMarket(market string, now time.Time) string {
	switch strings.ToUpper(strings.TrimSpace(market)) {
	case "US":
		loc, err := time.LoadLocation("America/New_York")
		if err != nil {
			return now.UTC().Format("2006-01-02")
		}
		return now.In(loc).Format("2006-01-02")
	default:
		return now.In(shanghaiLocation()).Format("2006-01-02")
	}
}

func marketFromCode(code string) string {
	parts := strings.Split(strings.ToUpper(strings.TrimSpace(code)), ".")
	if len(parts) < 2 {
		return ""
	}
	switch parts[len(parts)-1] {
	case "HK":
		return "HK"
	case "US":
		return "US"
	case "SH", "SZ", "SS":
		return "CN"
	default:
		return ""
	}
}

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
	if w != nil && strings.TrimSpace(w.ReportDate) != "" {
		return strings.TrimSpace(w.ReportDate)
	}
	market := ""
	if w != nil {
		market = strings.ToUpper(strings.TrimSpace(w.Market))
	}
	if market == "" {
		market = marketFromCode(code)
	}
	return CalendarDateForMarket(market, timeNow())
}
