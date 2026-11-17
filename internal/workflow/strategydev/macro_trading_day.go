package strategydev

import (
	"context"
	"fmt"
)

// MacroTradingDayChecker validates exchange calendar days (e.g. GeeGooBot checkTradingDay).
// When nil, macro cutoff may accept vendor placeholder bars on weekends/holidays — production should inject this.
type MacroTradingDayChecker interface {
	IsExchangeTradingDay(ctx context.Context, code, calendarDayYYYYMMDD string) (bool, error)
}

func macroBarEligibleTradingDay(ctx context.Context, checker MacroTradingDayChecker, code, calendarDay string, requireChecker bool) (bool, error) {
	if checker == nil {
		if requireChecker {
			return false, fmt.Errorf("trading day checker required but not configured")
		}
		return true, nil
	}
	ok, err := checker.IsExchangeTradingDay(ctx, code, calendarDay)
	if err != nil {
		return false, err
	}
	return ok, nil
}