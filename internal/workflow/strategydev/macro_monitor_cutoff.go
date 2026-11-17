package strategydev

import (
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// MacroMonitorCutoff is the latest completed regular-session daily bar used for macro lab-detail.
type MacroMonitorCutoff struct {
	AsOf         time.Time
	Close        float64
	CalendarDay  string // YYYY-MM-DD in session location
	DisplayTime  string // Asia/Shanghai formatted for anchor_time parity
}

// resolveMacroMonitorCutoffFromDailyBars picks the newest daily bar whose session day is complete at now.
// Fail-closed (ok=false) when no spec, no bars, or no completed session.
func resolveMacroMonitorCutoffFromDailyBars(code string, now time.Time, dailyBars []any) (MacroMonitorCutoff, bool) {
	spec, ok := macroSessionSpecForCode(code)
	if !ok || len(dailyBars) == 0 || now.IsZero() {
		return MacroMonitorCutoff{}, false
	}
	type candidate struct {
		calDay time.Time
		close  float64
		barTS  time.Time
	}
	var best *candidate
	for _, raw := range dailyBars {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ts := barTimeUnix(row)
		if ts.IsZero() {
			continue
		}
		calDay := calendarDateInLoc(ts, spec.loc)
		if calDay.IsZero() || !sessionDayComplete(now, calDay, spec) {
			continue
		}
		c := candidate{
			calDay: calDay,
			close:  slots.FloatAny(row["close"]),
			barTS:  ts,
		}
		if best == nil || calDay.After(best.calDay) {
			best = &c
		}
	}
	if best == nil {
		return MacroMonitorCutoff{}, false
	}
	asOf := sessionCloseInstant(best.calDay, spec)
	shLoc, _ := time.LoadLocation("Asia/Shanghai")
	display := asOf.In(shLoc).Format("2006-01-02 15:04") + " (Asia/Shanghai)"
	return MacroMonitorCutoff{
		AsOf:        asOf,
		Close:       best.close,
		CalendarDay: best.calDay.In(spec.loc).Format("2006-01-02"),
		DisplayTime: display,
	}, true
}

func calendarDateInLoc(ts time.Time, loc *time.Location) time.Time {
	if ts.IsZero() || loc == nil {
		return time.Time{}
	}
	d := ts.In(loc)
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
}

func applyMacroMonitorCutoffToDetail(out *LabEpisodeDetail, cut MacroMonitorCutoff) {
	if out == nil {
		return
	}
	out.MonitorQuoteCutoff = cut.AsOf.UTC().Format(time.RFC3339)
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