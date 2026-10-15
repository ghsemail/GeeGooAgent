package strategydev

import (
	"strings"
	"time"
)

// barTimeUnix parses OHLCV bar timestamps for price analysis (Asia/Shanghai wall clock).
func barTimeUnix(row map[string]any) time.Time {
	if row == nil {
		return time.Time{}
	}
	loc, locErr := time.LoadLocation("Asia/Shanghai")
	if locErr != nil {
		loc = time.UTC
	}
	for _, key := range []string{"time", "datetime", "date", "timestamp", "trade_date"} {
		if v := row[key]; v != nil {
			if ts := parseBarTimeValue(v, loc); !ts.IsZero() {
				return ts
			}
		}
	}
	return time.Time{}
}

func parseBarTimeValue(v any, loc *time.Location) time.Time {
	switch t := v.(type) {
	case string:
		return parseBarTimeString(strings.TrimSpace(t), loc)
	case float64:
		return parseBarTimeNumber(t, loc)
	case int:
		return parseBarTimeNumber(float64(t), loc)
	case int64:
		return parseBarTimeNumber(float64(t), loc)
	default:
		return time.Time{}
	}
}

func parseBarTimeString(s string, loc *time.Location) time.Time {
	if s == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339, s); err == nil {
		return parsed
	}
	if len(s) == 8 && isAllDigits(s) {
		if parsed, err := time.ParseInLocation("20060102", s, loc); err == nil {
			return parsed
		}
	}
	for _, layout := range []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006/01/02 15:04:05",
		"2006/01/02 15:04",
		"2006-01-02 15:04",
		"2006-01-02",
		"2006/01/02",
	} {
		if parsed, err := time.ParseInLocation(layout, s, loc); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func parseBarTimeNumber(v float64, loc *time.Location) time.Time {
	iv := int64(v)
	if iv >= 19900101 && iv <= 21001231 {
		y := int(iv / 10000)
		m := int(iv / 100 % 100)
		d := int(iv % 100)
		if m >= 1 && m <= 12 && d >= 1 && d <= 31 {
			return time.Date(y, time.Month(m), d, 0, 0, 0, 0, loc)
		}
	}
	sec := iv
	if sec > 1_000_000_000_000 {
		sec /= 1000
	}
	if sec > 1_000_000_000 {
		return time.Unix(sec, 0)
	}
	return time.Time{}
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
