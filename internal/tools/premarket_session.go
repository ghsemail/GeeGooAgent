package tools

import (
	"strings"
	"time"
)

// premarketAlreadyReportedForSession reports whether MCP premarket rows satisfy
// idempotency for the workflow session date. US symbols only count when the row
// was created on the same local calendar day as sessionDate.
func premarketAlreadyReportedForSession(code, sessionDate string, reports []map[string]any) bool {
	if len(reports) == 0 {
		return false
	}
	if !strings.HasSuffix(strings.ToUpper(strings.TrimSpace(code)), ".US") {
		return true
	}
	sessionDate = strings.TrimSpace(sessionDate)
	if sessionDate == "" {
		return true
	}
	loc := time.Local
	session, err := time.ParseInLocation("2006-01-02", sessionDate, loc)
	if err != nil {
		return true
	}
	for _, row := range reports {
		created := parseReportTimestamp(row["created_at"])
		if created.IsZero() {
			continue
		}
		c := created.In(loc)
		if c.Year() == session.Year() && c.YearDay() == session.YearDay() {
			return true
		}
	}
	return false
}

func parseReportTimestamp(v any) time.Time {
	s, _ := v.(string)
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
