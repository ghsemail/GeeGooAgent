package strategydev

import "context"

// MacroTradingDayChecker validates exchange calendar days (e.g. GeeGooBot checkTradingDay).
// When nil, macro cutoff may accept vendor placeholder bars on weekends/holidays — production should inject this.
type MacroTradingDayChecker interface {
	IsExchangeTradingDay(ctx context.Context, code, calendarDayYYYYMMDD string) (bool, error)
}

func macroBarEligibleTradingDay(ctx context.Context, checker MacroTradingDayChecker, code, calendarDay string) bool {
	if checker == nil {
		return true
	}
	ok, err := checker.IsExchangeTradingDay(ctx, code, calendarDay)
	if err != nil {
		return false
	}
	return ok
}