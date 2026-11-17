package strategydev

import (
	"strings"
	"time"
)

// macroSessionSpec defines when a calendar day's regular session is considered
// complete for macro monitor quote cutoff (exchange-local wall clock).
// Holidays are inferred from missing daily bars, not a local holiday table.
type macroSessionSpec struct {
	loc       *time.Location
	closeHour int
	closeMin  int
	// cryptoVendorLabel: calendar day from supplier daily bar label (not UTC/SH midnight session gate).
	cryptoVendorLabel bool
}

func macroSessionSpecForCode(code string) (macroSessionSpec, bool) {
	code = strings.TrimSpace(code)
	if code == "" {
		return macroSessionSpec{}, false
	}
	parts := strings.Split(code, ".")
	if len(parts) < 2 {
		return macroSessionSpec{}, false
	}
	suffix := strings.ToUpper(parts[len(parts)-1])
	switch suffix {
	case "SH", "SZ", "SS":
		loc, _ := time.LoadLocation("Asia/Shanghai")
		return macroSessionSpec{loc: loc, closeHour: 15, closeMin: 0}, true
	case "HK":
		loc, _ := time.LoadLocation("Asia/Hong_Kong")
		return macroSessionSpec{loc: loc, closeHour: 16, closeMin: 0}, true
	case "US", "NYB":
		loc, _ := time.LoadLocation("America/New_York")
		return macroSessionSpec{loc: loc, closeHour: 16, closeMin: 0}, true
	case "CRYPTO":
		return macroSessionSpec{loc: time.UTC, cryptoVendorLabel: true}, true
	default:
		return macroSessionSpec{}, false
	}
}

func sessionCloseInstant(calDay time.Time, spec macroSessionSpec) time.Time {
	if spec.loc == nil {
		return time.Time{}
	}
	if spec.cryptoVendorLabel {
		// Nominal end-of-label-day for JSON metadata only (not an exchange 23:59 gate).
		d := calDay.In(time.UTC)
		return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)
	}
	d := calDay.In(spec.loc)
	return time.Date(d.Year(), d.Month(), d.Day(), spec.closeHour, spec.closeMin, 0, 0, spec.loc)
}

// sessionDayComplete is true when now is at or after the exchange close for calDay.
// Half-days and exchange holiday calendars are not modeled here (missing bar only).
func sessionDayComplete(now, calDay time.Time, spec macroSessionSpec) bool {
	if spec.cryptoVendorLabel {
		// Vendor label days: completion is decided by presence of a complete daily bar in cutoff scan, not UTC 23:59.
		return true
	}
	closeAt := sessionCloseInstant(calDay, spec)
	if closeAt.IsZero() {
		return false
	}
	return !now.Before(closeAt)
}