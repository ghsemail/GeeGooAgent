package args

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
)

// PremarketReportDateForPostmarket returns the report_date used to load stock
// premarket reports during post-market workflow. CN/HK: same session day.
// US postmarket at 05:00 Asia/Shanghai is still the previous US/Eastern day,
// so when sessionDate is that Eastern date, premarket is the same calendar day.
func PremarketReportDateForPostmarket(code, sessionDate string) string {
	sessionDate = strings.TrimSpace(sessionDate)
	if sessionDate == "" {
		if strings.HasSuffix(strings.ToUpper(strings.TrimSpace(code)), ".US") {
			return CalendarDateForMarket("US", timeNow())
		}
		return CalendarDateForMarket("", timeNow())
	}
	return sessionDate
}

// PostmarketPremarketLookupArg builds get_stock_daily_reports args for post-market
// premarket hydration (US session_date is already US/Eastern).
func PostmarketPremarketLookupArg(w *memory.PreMarketWorking) map[string]any {
	code := w.CurrentStock
	sessionDate := ReportDateFor(w, code)
	return map[string]any{
		"code":        code,
		"report_date": PremarketReportDateForPostmarket(code, sessionDate),
	}
}
