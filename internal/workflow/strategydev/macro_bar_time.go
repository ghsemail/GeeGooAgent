package strategydev

import (
	"strings"
	"time"
)

func calendarDateInLoc(ts time.Time, loc *time.Location) time.Time {
	if ts.IsZero() || loc == nil {
		return time.Time{}
	}
	d := ts.In(loc)
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
}

// macroBarCalendarDay maps a daily bar row to the exchange-session calendar label in spec.loc.
// Date-only / midnight strings without offset are treated as session date labels (not Shanghai instants).
func macroBarCalendarDay(row map[string]any, spec macroSessionSpec) (time.Time, bool) {
	if row == nil || spec.loc == nil {
		return time.Time{}, false
	}
	if day, ok := macroBarDateLabelInLoc(row, spec.loc); ok {
		return day, true
	}
	ts := barTimeUnix(row)
	if ts.IsZero() {
		return time.Time{}, false
	}
	if ts.Location() == time.UTC && ts.Hour() == 0 && ts.Minute() == 0 && ts.Second() == 0 {
		// Instant UTC midnight often encodes a calendar label for non-CN dailies.
		return time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, spec.loc), true
	}
	return calendarDateInLoc(ts, spec.loc), true
}

func macroBarDateLabelInLoc(row map[string]any, loc *time.Location) (time.Time, bool) {
	for _, key := range []string{"time", "datetime", "date", "trade_date"} {
		v := row[key]
		if v == nil {
			continue
		}
		s, ok := v.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if parsed, err := time.Parse(time.RFC3339, s); err == nil {
			d := parsed.In(loc)
			return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc), true
		}
		for _, layout := range []string{
			"2006-01-02",
			"2006/01/02",
			"20060102",
			"2006-01-02 00:00:00",
			"2006-01-02T00:00:00",
			"2006/01/02 00:00:00",
		} {
			if t, err := time.ParseInLocation(layout, s, loc); err == nil {
				return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc), true
			}
		}
	}
	return time.Time{}, false
}

// indexDailyBarByCalendarDayExact returns the bar index for an exact session calendar day only (no nearest fallback).
func indexDailyBarByCalendarDayExact(dailyBars []any, day string, spec macroSessionSpec) int {
	day = strings.TrimSpace(day)
	if day == "" || len(dailyBars) == 0 {
		return -1
	}
	want, err := time.ParseInLocation("2006-01-02", day, spec.loc)
	if err != nil {
		return -1
	}
	for i, raw := range dailyBars {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		cal, ok := macroBarCalendarDay(row, spec)
		if !ok || cal.IsZero() {
			continue
		}
		if cal.Equal(want) {
			return i
		}
	}
	return -1
}