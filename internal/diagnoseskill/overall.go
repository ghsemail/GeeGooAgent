package diagnoseskill

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

// MacroOverallPoolRow is one pool line in macro_overall skill request.
type MacroOverallPoolRow struct {
	PoolWorkflow string
	JEVDirection string
	JEVReason    string
}

// MacroOverallRunInput is validated payload for diagnose_macro_overall.
type MacroOverallRunInput struct {
	Date         string
	Pools        []MacroOverallPoolRow
	JEV          *strategydev.JevClient
	SessionID    string
	RunID        string
	MinReasonLen int
}

// ValidateMacroOverallInput checks macro_overall pools (1–7, unique, legal).
func ValidateMacroOverallInput(in MacroOverallRunInput) error {
	date := strings.TrimSpace(in.Date)
	if !isoDateRE.MatchString(date) {
		return fmt.Errorf("invalid date")
	}
	if len(in.Pools) == 0 {
		return fmt.Errorf("pools required")
	}
	if len(in.Pools) > 7 {
		return fmt.Errorf("too many pools")
	}
	minLen := in.MinReasonLen
	if minLen <= 0 {
		minLen = 20
	}
	seen := map[string]struct{}{}
	for _, p := range in.Pools {
		wf := strings.TrimSpace(p.PoolWorkflow)
		if _, ok := poolWorkflows[wf]; !ok {
			return fmt.Errorf("invalid pool_workflow")
		}
		if _, dup := seen[wf]; dup {
			return fmt.Errorf("duplicate pool_workflow")
		}
		seen[wf] = struct{}{}
		dir := strings.TrimSpace(strings.ToLower(p.JEVDirection))
		if dir != "long" && dir != "short" && dir != "neutral" {
			return fmt.Errorf("invalid jev_direction for %s", wf)
		}
		if len(strings.TrimSpace(p.JEVReason)) < minLen {
			return fmt.Errorf("jev_reason too short for %s", wf)
		}
	}
	return nil
}

// RunOverall executes macro overall JEV synthesis; does not PUT canonical tables.
func RunOverall(ctx context.Context, in MacroOverallRunInput) (Result, error) {
	if err := ValidateMacroOverallInput(in); err != nil {
		return nil, err
	}
	var snapshots []strategydev.MacroOverallPoolInput
	for _, p := range in.Pools {
		snapshots = append(snapshots, strategydev.MacroOverallPoolInput{
			PoolWorkflow: strings.TrimSpace(p.PoolWorkflow),
			JEVDirection: strings.TrimSpace(strings.ToLower(p.JEVDirection)),
			JEVReason:    strings.TrimSpace(p.JEVReason),
		})
	}
	jevCtx, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	rec := strategydev.ResolveMacroOverallJEV(jevCtx, in.JEV, strings.TrimSpace(in.Date), snapshots)
	if !rec.OK {
		return nil, fmt.Errorf("%s", strings.TrimSpace(rec.Error))
	}
	ledgerDir := DirectionToLedger(string(rec.Direction))
	if ledgerDir != "long" && ledgerDir != "short" && ledgerDir != "neutral" {
		return nil, fmt.Errorf("overall jev_direction invalid")
	}
	reason := strings.TrimSpace(rec.RecommendedReason)
	minLen := in.MinReasonLen
	if minLen <= 0 {
		minLen = 20
	}
	if len(reason) < minLen {
		return nil, fmt.Errorf("overall jev_reason too short")
	}
	choices := poolChoicesForResult(rec.Choices)
	if len(choices) == 0 {
		return nil, fmt.Errorf("overall jev choices missing")
	}
	return Result{
		"date":          strings.TrimSpace(in.Date),
		"detail_ref":    "diagnose_macro_overall",
		"jev_direction": ledgerDir,
		"jev_reason":    reason,
		"confidence":    rec.Confidence,
		"choices":       choices,
	}, nil
}