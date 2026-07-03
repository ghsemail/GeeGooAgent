// Package verify performs cutover acceptance checks on persisted pre-market
// reports. It is deliberately pure (no I/O) so it can be unit-tested with
// fixtures and reused by `geegoo verify` against live API responses.
package verify

import (
	"fmt"
	"strings"
)

// FieldCheck is the outcome of one field-completeness check.
type FieldCheck struct {
	Name   string
	Passed bool
	Detail string
}

// ReportCard is the verdict for one pre-market report record.
type ReportCard struct {
	Code     string
	ReportID string
	Checks   []FieldCheck
	Passed   bool
}

// Summary returns a one-line verdict.
func (c ReportCard) Summary() string {
	pass, fail := 0, 0
	for _, ch := range c.Checks {
		if ch.Passed {
			pass++
		} else {
			fail++
		}
	}
	verdict := "PASS"
	if !c.Passed {
		verdict = "FAIL"
	}
	return fmt.Sprintf("%s code=%s report=%s checks=%d pass=%d fail=%d",
		verdict, c.Code, c.ReportID, len(c.Checks), pass, fail)
}

var (
	validResults     = map[string]bool{"long": true, "short": true, "neutral": true}
	validConfidences = map[string]bool{"high": true, "medium": true, "low": true, "review_required": true}
	validSuggestions = map[string]bool{"buy": true, "sell": true, "hold": true, "watch_long": true, "reduce_or_avoid": true}
)

// VerifyReport checks one createPreMarketReport record for required fields and
// value enums, matching rules/report-format.md and the Hermes parity checklist.
func VerifyReport(report map[string]any) ReportCard {
	card := ReportCard{
		Code:     strField(report, "code"),
		ReportID: strField(report, "report_id"),
	}
	checks := []FieldCheck{
		hasNonEmpty("bot_id", report),
		hasNonEmpty("bot_name", report),
		hasNonEmpty("bot_type", report),
		hasNonEmpty("stock_name", report),
		hasEnum("result", report, validResults),
		hasEnum("confidence", report, validConfidences),
		hasEnum("suggestion", report, validSuggestions),
		hasMinLength("reason", report, 80),
		hasNonEmpty("report", report),
		hasEvidenceRefs(report),
	}
	card.Checks = checks
	card.Passed = true
	for _, ch := range checks {
		if !ch.Passed {
			card.Passed = false
			break
		}
	}
	return card
}

// VerifyReports runs VerifyReport over a slice and returns the cards plus an
// overall pass bool (true only if every report passes).
func VerifyReports(reports []map[string]any) []ReportCard {
	out := make([]ReportCard, 0, len(reports))
	for _, r := range reports {
		out = append(out, VerifyReport(r))
	}
	return out
}

// AllPass reports whether every card passed.
func AllPass(cards []ReportCard) bool {
	for _, c := range cards {
		if !c.Passed {
			return false
		}
	}
	return true
}

// CompletenessMatrix returns per-field non-empty rates across cards, useful
// for the "field completeness matrix" requirement in the migration checklist.
func CompletenessMatrix(cards []ReportCard) map[string]float64 {
	if len(cards) == 0 {
		return nil
	}
	totals := map[string]int{}
	for _, c := range cards {
		for _, ch := range c.Checks {
			totals[ch.Name]++
			if ch.Passed {
				totals[ch.Name+"_pass"]++
			}
		}
	}
	out := map[string]float64{}
	for name, total := range totals {
		if strings.HasSuffix(name, "_pass") {
			continue
		}
		pass := totals[name+"_pass"]
		if total > 0 {
			out[name] = float64(pass) / float64(total)
		}
	}
	return out
}

func strField(m map[string]any, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}

func hasNonEmpty(field string, m map[string]any) FieldCheck {
	v := strings.TrimSpace(strField(m, field))
	if v != "" {
		return FieldCheck{Name: field, Passed: true, Detail: v}
	}
	return FieldCheck{Name: field, Passed: false, Detail: "empty"}
}

func hasEnum(field string, m map[string]any, allowed map[string]bool) FieldCheck {
	v := strings.TrimSpace(strField(m, field))
	if v == "" {
		return FieldCheck{Name: field, Passed: false, Detail: "empty"}
	}
	if allowed[v] {
		return FieldCheck{Name: field, Passed: true, Detail: v}
	}
	return FieldCheck{Name: field, Passed: false, Detail: fmt.Sprintf("invalid enum %q", v)}
}

func hasMinLength(field string, m map[string]any, min int) FieldCheck {
	v := strField(m, field)
	if len(strings.TrimSpace(v)) >= min {
		return FieldCheck{Name: field, Passed: true, Detail: fmt.Sprintf("%d chars", len(v))}
	}
	return FieldCheck{Name: field, Passed: false, Detail: fmt.Sprintf("only %d chars (need %d)", len(v), min)}
}

func hasEvidenceRefs(m map[string]any) FieldCheck {
	v, ok := m["evidence_refs"]
	if !ok || v == nil {
		return FieldCheck{Name: "evidence_refs", Passed: false, Detail: "missing"}
	}
	switch refs := v.(type) {
	case []any:
		if len(refs) > 0 {
			return FieldCheck{Name: "evidence_refs", Passed: true, Detail: fmt.Sprintf("%d refs", len(refs))}
		}
	case []string:
		if len(refs) > 0 {
			return FieldCheck{Name: "evidence_refs", Passed: true, Detail: fmt.Sprintf("%d refs", len(refs))}
		}
	}
	return FieldCheck{Name: "evidence_refs", Passed: false, Detail: "empty"}
}
