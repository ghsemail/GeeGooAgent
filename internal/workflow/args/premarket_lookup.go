package args

import (
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
)

// PremarketReportDateForPostmarket returns the report_date used to load stock
// premarket reports during post-market workflow. CN/HK: same session day; US:
// previous calendar day (premarket is generated the prior evening CST).
func PremarketReportDateForPostmarket(code, sessionDate string) string {
	sessionDate = strings.TrimSpace(sessionDate)
	if sessionDate == "" {
		sessionDate = timeNow().Format("2006-01-02")
	}
	if !strings.HasSuffix(strings.ToUpper(strings.TrimSpace(code)), ".US") {
		return sessionDate
	}
	t, err := time.ParseInLocation("2006-01-02", sessionDate, timeNow().Location())
	if err != nil {
		return sessionDate
	}
	return t.AddDate(0, 0, -1).Format("2006-01-02")
}

// PostmarketPremarketLookupArg builds get_stock_daily_reports args for post-market
// premarket hydration (US looks back one day).
func PostmarketPremarketLookupArg(w *memory.PreMarketWorking) map[string]any {
	code := w.CurrentStock
	sessionDate := ReportDateFor(w, code)
	return map[string]any{
		"code":        code,
		"report_date": PremarketReportDateForPostmarket(code, sessionDate),
	}
}
