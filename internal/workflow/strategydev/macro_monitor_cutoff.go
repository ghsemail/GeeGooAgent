package strategydev

import (
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

// resolveMacroMonitorCutoffFromDailyBars picks the newest daily bar whose session day is complete at now.
// Fail-closed (ok=false) when no spec, no bars, or no completed session.
func resolveMacroMonitorCutoffFromDailyBars(code string, now time.Time, dailyBars []any) (MacroMonitorCutoff, bool) {
	spec, ok := macroSessionSpecForCode(code)
	if !ok || len(dailyBars) == 0 || now.IsZero() {
		return MacroMonitorCutoff{}, false
	}
	todayCal := calendarDateInLoc(now, spec.loc)
	type candidate struct {
		calDay time.Time
		close  float64
		barTS  time.Time
		barRaw string
	}
	var best *candidate
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
		if !sessionDayComplete(now, calDay, spec) {
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