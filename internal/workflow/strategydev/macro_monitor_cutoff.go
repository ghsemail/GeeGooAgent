package strategydev

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// MacroMonitorCutoff is the latest completed regular-session daily bar used for macro lab-detail.
type MacroMonitorCutoff struct {
	SessionClose time.Time // exchange-local session close instant
	Close        float64
	CalendarDay  string // YYYY-MM-DD in session location
	DisplayTime  string // Asia/Shanghai formatted for anchor_time parity
	BarTimeRaw   string // supplier bar timestamp string when present
	BarTime      time.Time
}

func labDetailNow(in LabDetailInput) time.Time {
	if in.Now != nil {
		if t := in.Now(); !t.IsZero() {
			return t
		}
	}
	return time.Now()
}

// resolveMacroMonitorCutoffFromDailyBars picks the newest daily bar whose session day is complete at now.
// Fail-closed (ok=false) when no spec, no bars, or no completed session.
func resolveMacroMonitorCutoffFromDailyBars(code string, now time.Time, dailyBars []any) (MacroMonitorCutoff, bool) {
	return resolveMacroMonitorCutoffFromDailyBarsWithChecker(context.Background(), nil, false, code, now, dailyBars)
}

func resolveDailyMacroCutoff(ctx context.Context, in LabDetailInput, dailyBars []any) (MacroMonitorCutoff, bool, string) {
	require := labDetailDailyMacro(in) && in.RequireTradingDayChecker
	cut, ok := resolveMacroMonitorCutoffFromDailyBarsWithChecker(ctx, in.TradingDayChecker, require, in.Code, labDetailNow(in), dailyBars)
	if ok {
		return cut, true, ""
	}
	if require {
		if in.TradingDayChecker == nil {
			return MacroMonitorCutoff{}, false, "macro_trading_day_checker_unavailable"
		}
		return MacroMonitorCutoff{}, false, "macro_trading_day_check_failed"
	}
	return MacroMonitorCutoff{}, false, "macro_monitor_cutoff_unavailable"
}

func resolveMacroMonitorCutoffFromDailyBarsWithChecker(
	ctx context.Context,
	checker MacroTradingDayChecker,
	requireChecker bool,
	code string,
	now time.Time,
	dailyBars []any,
) (MacroMonitorCutoff, bool) {
	spec, ok := macroSessionSpecForCode(code)
	if !ok || len(dailyBars) == 0 || now.IsZero() {
		return MacroMonitorCutoff{}, false
	}
	if requireChecker && checker == nil {
		return MacroMonitorCutoff{}, false
	}
	todayCal := macroMonitorUpperCalendarDay(now, spec, dailyBars)
	type candidate struct {
		calDay time.Time
		close  float64
		barTS  time.Time
		barRaw string
	}
	var best *candidate
	var checkErr error
	for _, raw := range dailyBars {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		calDay, ok := macroBarCalendarDay(row, spec)
		if !ok || calDay.IsZero() {
			continue
		}
		if calDay.After(todayCal) {
			continue
		}
		if spec.cryptoVendorLabel {
			if !cryptoVendorDailyBarComplete(row) {
				continue
			}
		} else if !sessionDayComplete(now, calDay, spec) {
			continue
		}
		dayLabel := calDay.In(spec.loc).Format("2006-01-02")
		eligible, err := macroBarEligibleTradingDay(ctx, checker, code, dayLabel, requireChecker)
		if err != nil {
			checkErr = err
			continue
		}
		if !eligible {
			continue
		}
		closePx := slots.FloatAny(row["close"])
		if closePx <= 0 {
			continue
		}
		barTS, barRaw := macroBarTimestamp(row, spec)
		c := candidate{
			calDay: calDay,
			close:  closePx,
			barTS:  barTS,
			barRaw: barRaw,
		}
		if best == nil || calDay.After(best.calDay) {
			best = &c
		}
	}
	if best == nil {
		if checkErr != nil && requireChecker {
			return MacroMonitorCutoff{}, false
		}
		return MacroMonitorCutoff{}, false
	}
	sessionClose := sessionCloseInstant(best.calDay, spec)
	shLoc, _ := time.LoadLocation("Asia/Shanghai")
	display := sessionClose.In(shLoc).Format("2006-01-02 15:04") + " (Asia/Shanghai)"
	return MacroMonitorCutoff{
		SessionClose: sessionClose,
		Close:        best.close,
		CalendarDay:  best.calDay.In(spec.loc).Format("2006-01-02"),
		DisplayTime:  display,
		BarTimeRaw:   best.barRaw,
		BarTime:      best.barTS,
	}, true
}

func macroBarTimestamp(row map[string]any, spec macroSessionSpec) (time.Time, string) {
	for _, key := range []string{"time", "datetime", "date", "trade_date"} {
		if v := row[key]; v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				s = strings.TrimSpace(s)
				if day, ok := macroBarDateLabelInLoc(row, spec.loc); ok {
					return day, s
				}
				ts := barTimeUnix(row)
				return ts, s
			}
		}
	}
	return time.Time{}, ""
}

func applyMacroMonitorCutoffToDetail(out *LabEpisodeDetail, cut MacroMonitorCutoff) {
	if out == nil {
		return
	}
	sessionRFC := cut.SessionClose.UTC().Format(time.RFC3339)
	out.MonitorSessionClose = sessionRFC
	out.MonitorQuoteBarTime = strings.TrimSpace(cut.BarTimeRaw)
	if cut.Close > 0 {
		out.MonitorQuoteBarClose = cut.Close
	}
	// Back-compat alias: monitor_quote_cutoff* mean session close semantics, not last trade print.
	out.MonitorQuoteCutoff = sessionRFC
	out.MonitorQuoteCutoffDay = cut.CalendarDay
	out.MonitorQuoteCutoffDisplay = strings.TrimSpace(cut.DisplayTime)
	if out.PriceAnalysis != nil {
		out.PriceAnalysis.AnchorTime = cut.DisplayTime
		out.PriceAnalysis.AnchorClose = cut.Close
		if cut.Close > 0 {
			out.PriceAnalysis.SpotPrice = cut.Close
		}
	}
}

// formatMacroMonitorCutoffDocuments FE contract note in errors.
func formatMacroMonitorCutoffDocuments(code string) string {
	return fmt.Sprintf("macro monitor cutoff unavailable for %s (no completed session bar)", strings.TrimSpace(code))
}

func resolveDailyMacroMonitorCutoff(code string, now time.Time, dailyBars []any) (MacroMonitorCutoff, bool) {
	return resolveMacroMonitorCutoffFromDailyBars(code, now, dailyBars)
}

func macroMonitorUpperCalendarDay(now time.Time, spec macroSessionSpec, dailyBars []any) time.Time {
	if spec.cryptoVendorLabel {
		var max time.Time
		for _, raw := range dailyBars {
			row, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			cal, ok := macroBarCalendarDay(row, spec)
			if !ok || cal.IsZero() {
				continue
			}
			if cal.After(max) {
				max = cal
			}
		}
		if !max.IsZero() {
			return max
		}
	}
	return calendarDateInLoc(now, spec.loc)
}

func cryptoVendorDailyBarComplete(row map[string]any) bool {
	if row == nil {
		return false
	}
	closePx := slots.FloatAny(row["close"])
	if closePx <= 0 {
		return false
	}
	openPx := slots.FloatAny(row["open"])
	highPx := slots.FloatAny(row["high"])
	lowPx := slots.FloatAny(row["low"])
	if openPx > 0 && highPx > 0 && lowPx > 0 {
		return true
	}
	// Supplier may omit OHLC legs; close>0 still counts as a published vendor daily for the label.
	return closePx > 0
}