package eval

import (
	"encoding/json"
	"time"
)

// TurnPlanTurnResult is one turn outcome for dashboard / API responses.
type TurnPlanTurnResult struct {
	TurnID  string `json:"turn_id"`
	Message string `json:"message"`
	Passed  bool   `json:"passed"`
	Detail  string `json:"detail"`
}

// TurnPlanRunReport is the API/CLI report for a turn_plan suite run.
type TurnPlanRunReport struct {
	CaseID     string               `json:"case_id,omitempty"`
	Passed     int                  `json:"passed"`
	Failed     int                  `json:"failed"`
	Total      int                  `json:"total"`
	AllPass    bool                 `json:"all_pass"`
	DurationMs int64                `json:"duration_ms"`
	Results    []TurnPlanTurnResult `json:"results"`
}

// ParseTurnPlanSuite decodes options_json or a full suite payload.
func ParseTurnPlanSuite(raw []byte) (TurnPlanSuite, error) {
	var suite TurnPlanSuite
	if err := json.Unmarshal(raw, &suite); err != nil {
		return TurnPlanSuite{}, err
	}
	return suite, nil
}

// SuiteFromOptions maps dashboard case options to a TurnPlanSuite.
func SuiteFromOptions(opts map[string]any) (TurnPlanSuite, error) {
	raw, err := json.Marshal(opts)
	if err != nil {
		return TurnPlanSuite{}, err
	}
	return ParseTurnPlanSuite(raw)
}

// RunTurnPlanReport executes the suite and returns a structured report.
func RunTurnPlanReport(suite TurnPlanSuite) TurnPlanRunReport {
	start := time.Now()
	results := RunTurnPlanSuite(suite)
	turnMsg := map[string]string{}
	for _, turn := range suite.Turns {
		turnMsg[turn.ID] = turn.Message
	}
	out := make([]TurnPlanTurnResult, 0, len(results))
	passed := 0
	for _, r := range results {
		if r.Passed {
			passed++
		}
		out = append(out, TurnPlanTurnResult{
			TurnID:  r.TurnID,
			Message: turnMsg[r.TurnID],
			Passed:  r.Passed,
			Detail:  r.Detail,
		})
	}
	total := len(results)
	return TurnPlanRunReport{
		Passed:     passed,
		Failed:     total - passed,
		Total:      total,
		AllPass:    passed == total && total > 0,
		DurationMs: time.Since(start).Milliseconds(),
		Results:    out,
	}
}
