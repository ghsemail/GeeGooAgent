package diagnoseskill

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

var (
	isoDateRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	poolWorkflows = map[string]struct{}{
		"cn_a": {}, "hk": {}, "us": {}, "commodity_fx": {}, "bond": {}, "crypto": {}, "volatility": {},
	}
)

// MacroPoolRunInput is validated workflow payload for diagnose_macro_pool.
type MacroPoolRunInput struct {
	Date             string
	PoolWorkflow     string
	SourceBatchID    string
	Title            string
	ExpectedSymbols  []string
	Symbols          []MacroPoolSymbolRow
	JEV              *strategydev.JevClient
	SessionID        string
	RunID            string
	MinReasonLen     int
}

// MacroPoolSymbolRow is one symbol line in the pool skill request.
type MacroPoolSymbolRow struct {
	Symbol       string
	ResultID     string
	JEVDirection string
	JEVReason    string
	Confidence   float64
}

// ValidateMacroPoolInput checks workflow-assembled pool payload (fail-closed).
func ValidateMacroPoolInput(in MacroPoolRunInput) error {
	date := strings.TrimSpace(in.Date)
	if !isoDateRE.MatchString(date) {
		return fmt.Errorf("invalid date")
	}
	wf := strings.TrimSpace(in.PoolWorkflow)
	if _, ok := poolWorkflows[wf]; !ok {
		return fmt.Errorf("invalid pool_workflow")
	}
	if strings.TrimSpace(in.SourceBatchID) == "" {
		return fmt.Errorf("source_batch_id required")
	}
	if len(in.ExpectedSymbols) == 0 {
		return fmt.Errorf("expected_symbols required")
	}
	if len(in.Symbols) == 0 {
		return fmt.Errorf("symbols required")
	}
	expSet := normalizeSymbolSet(in.ExpectedSymbols)
	if len(expSet) == 0 {
		return fmt.Errorf("expected_symbols empty")
	}
	gotSet := map[string]struct{}{}
	minLen := in.MinReasonLen
	if minLen <= 0 {
		minLen = 20
	}
	seen := map[string]struct{}{}
	for _, sym := range in.Symbols {
		code := strings.TrimSpace(sym.Symbol)
		if code == "" {
			return fmt.Errorf("symbol required")
		}
		upper := strings.ToUpper(code)
		if _, dup := seen[upper]; dup {
			return fmt.Errorf("duplicate symbol %s", code)
		}
		seen[upper] = struct{}{}
		gotSet[upper] = struct{}{}
		if strings.TrimSpace(sym.ResultID) == "" {
			return fmt.Errorf("result_id required for %s", code)
		}
		dir := strings.TrimSpace(strings.ToLower(sym.JEVDirection))
		if dir != "long" && dir != "short" && dir != "neutral" {
			return fmt.Errorf("invalid jev_direction for %s", code)
		}
		if len(strings.TrimSpace(sym.JEVReason)) < minLen {
			return fmt.Errorf("jev_reason too short for %s", code)
		}
	}
	if len(gotSet) != len(expSet) {
		return fmt.Errorf("symbols count %d != expected_symbols count %d", len(gotSet), len(expSet))
	}
	for upper := range expSet {
		if _, ok := gotSet[upper]; !ok {
			return fmt.Errorf("missing expected symbol %s", upper)
		}
	}
	for upper := range gotSet {
		if _, ok := expSet[upper]; !ok {
			return fmt.Errorf("unexpected symbol %s", upper)
		}
	}
	return nil
}

func normalizeSymbolSet(syms []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, s := range syms {
		code := strings.TrimSpace(s)
		if code == "" {
			continue
		}
		out[strings.ToUpper(code)] = struct{}{}
	}
	return out
}

// RunPool executes pool JEV synthesis; does not PUT canonical tables.
func RunPool(ctx context.Context, in MacroPoolRunInput) (Result, error) {
	if err := ValidateMacroPoolInput(in); err != nil {
		return nil, err
	}
	var snapshots []strategydev.MacroPoolSymbolInput
	var inputRefs []map[string]any
	for _, sym := range in.Symbols {
		dir := strings.TrimSpace(strings.ToLower(sym.JEVDirection))
		snapshots = append(snapshots, strategydev.MacroPoolSymbolInput{
			Symbol:       sym.Symbol,
			ResultID:     sym.ResultID,
			JEVDirection: dir,
			JEVReason:    sym.JEVReason,
			Confidence:   sym.Confidence,
		})
		inputRefs = append(inputRefs, map[string]any{
			"symbol":    strings.TrimSpace(sym.Symbol),
			"result_id": strings.TrimSpace(sym.ResultID),
		})
	}
	jevCtx, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	rec := strategydev.ResolveMacroPoolJEV(jevCtx, in.JEV, in.PoolWorkflow, in.Title, snapshots)
	if !rec.OK {
		return nil, fmt.Errorf("%s", strings.TrimSpace(rec.Error))
	}
	ledgerDir := DirectionToLedger(string(rec.Direction))
	if ledgerDir != "long" && ledgerDir != "short" && ledgerDir != "neutral" {
		return nil, fmt.Errorf("pool jev direction invalid")
	}
	reason := strings.TrimSpace(rec.RecommendedReason)
	if len(reason) < in.MinReasonLen && in.MinReasonLen > 0 {
		return nil, fmt.Errorf("pool jev_reason too short")
	}
	choices := poolChoicesForResult(rec.Choices)
	if len(choices) == 0 {
		return nil, fmt.Errorf("pool jev choices missing")
	}
	res := Result{
		"date":             strings.TrimSpace(in.Date),
		"pool_workflow":    strings.TrimSpace(in.PoolWorkflow),
		"source_batch_id":  strings.TrimSpace(in.SourceBatchID),
		"detail_ref":       "diagnose_macro_pool",
		"jev_direction":    ledgerDir,
		"jev_reason":       reason,
		"confidence":       rec.Confidence,
		"conflict":         rec.Conflict,
		"choices":          choices,
		"input_refs":       inputRefs,
	}
	return res, nil
}

// poolChoicesForResult mirrors emit lane JEV: [{label, probability}] for the current Choice.
func poolChoicesForResult(choices []strategydev.PriceAnalysisJEVChoice) []map[string]any {
	if len(choices) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(choices))
	for _, c := range choices {
		label := strings.TrimSpace(c.Label)
		if label == "" {
			continue
		}
		out = append(out, map[string]any{
			"label":       label,
			"probability": c.Probability,
		})
	}
	return out
}