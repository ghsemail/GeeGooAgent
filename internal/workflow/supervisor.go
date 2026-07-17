package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
)

// Verdict is the supervisor's overall acceptance decision.
type Verdict string

const (
	// VerdictPass: all checks passed; workflow considered complete.
	VerdictPass Verdict = "pass"
	// VerdictRecoverable: one or more steps can be re-run to fix gaps.
	VerdictRecoverable Verdict = "recoverable"
	// VerdictTerminal: a hard contract failure that re-running cannot fix
	// (e.g. create_pre_market_report returned a business error). Human attention needed.
	VerdictTerminal Verdict = "terminal"
)

// CheckResult is the outcome of one acceptance check.
type CheckResult struct {
	Name    string
	Type    string
	Passed  bool
	Detail  string
	Skipped bool
}

// SupervisorReport is the supervisor's verdict plus per-check detail.
type SupervisorReport struct {
	Verdict      Verdict
	Checks       []CheckResult
	MissingSteps []string // recoverable: named steps that should be re-run
	GeneratedAt  time.Time
}

// Summary returns a one-line human description.
func (r SupervisorReport) Summary() string {
	pass, fail := 0, 0
	for _, c := range r.Checks {
		if c.Skipped {
			continue
		}
		if c.Passed {
			pass++
		} else {
			fail++
		}
	}
	return fmt.Sprintf("verdict=%s checks=%d pass=%d fail=%d", r.Verdict, len(r.Checks), pass, fail)
}

// Check is one acceptance check spec (mirrors supervisor_checks.yaml entries).
type Check struct {
	Name          string
	Type          string
	ExpectPhase   string   `yaml:"expect_phase"`
	ForStatus     string   `yaml:"for_status"`
	Pattern       string   `yaml:"pattern"`
	RequireFields []string `yaml:"require_fields"`
	Required      []string `yaml:"required"`
}

// DefaultPreMarketChecks returns the pre_market acceptance checks as Go values.
// This mirrors skills/pre_market/supervisor_checks.yaml so P3 does not require
// a YAML parser dependency; the YAML file remains the source of truth for docs.
func DefaultPreMarketChecks() []Check {
	return []Check{
		{Name: "workflow_phase_done", Type: "stocks_status", ExpectPhase: "done"},
		{Name: "reported_stocks_local_md", Type: "file_exists", Pattern: "reports/{date}/{code}-premarket.md", ForStatus: "reported"},
		{Name: "reported_stocks_api_created", Type: "stocks_status", ForStatus: "reported", RequireFields: []string{"report_id"}},
		{Name: "reported_stocks_bot_fields", Type: "stocks_status", ForStatus: "reported", RequireFields: []string{"bot_id", "bot_name", "bot_type"}},
		{Name: "reported_stocks_api_payload", Type: "api_response", ForStatus: "reported",
			Required: []string{"code", "stock_name", "bot_id", "bot_name", "bot_type", "result", "confidence", "reason", "suggestion", "report"}},
	}
}

// SupervisorChecksForSkill returns acceptance checks for a workflow skill.
func SupervisorChecksForSkill(skill string) []Check {
	switch skill {
	case "intraday":
		return DefaultIntradayChecks()
	case "post_market":
		return DefaultPostMarketChecks()
	default:
		return DefaultPreMarketChecks()
	}
}

// DefaultIntradayChecks mirrors skills/intraday/supervisor_checks.yaml.
func DefaultIntradayChecks() []Check {
	return []Check{
		{Name: "intraday_reported", Type: "stocks_status", ForStatus: "reported", RequireFields: []string{"report_id"}},
		{Name: "intraday_local_md", Type: "file_exists", Pattern: "reports/{date}/{code}-intraday.md", ForStatus: "reported"},
	}
}

// DefaultPostMarketChecks mirrors skills/post_market/supervisor_checks.yaml.
func DefaultPostMarketChecks() []Check {
	return []Check{
		{Name: "workflow_phase_done", Type: "stocks_status", ExpectPhase: "done"},
		{Name: "post_market_local_md", Type: "file_exists", Pattern: "reports/{date}/{code}-postmarket.md", ForStatus: "reported"},
		{Name: "post_market_reported", Type: "stocks_status", ForStatus: "reported", RequireFields: []string{"report_id"}},
	}
}

// Engine runs acceptance checks against working memory.
type Engine struct {
	checks []Check
	// workspaceRoot resolves file_exists patterns.
	workspaceRoot string
}

// NewEngine creates a supervisor engine.
func NewEngine(workspaceRoot string, checks []Check) *Engine {
	if checks == nil {
		checks = DefaultPreMarketChecks()
	}
	return &Engine{checks: checks, workspaceRoot: workspaceRoot}
}

// Verify evaluates all checks and produces a report.
func (e *Engine) Verify(working *memory.PreMarketWorking, date string) SupervisorReport {
	report := SupervisorReport{GeneratedAt: time.Now().UTC()}
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	hardFail := false
	for _, c := range e.checks {
		res := e.runCheck(c, working, date)
		report.Checks = append(report.Checks, res)
		if !res.Skipped && !res.Passed && c.Type == "stocks_status" && c.ExpectPhase == "done" {
			// phase not done is recoverable, not terminal
		}
	}
	// Determine verdict.
	anyFail := false
	for _, c := range report.Checks {
		if c.Skipped {
			continue
		}
		if !c.Passed {
			anyFail = true
			// A failed stock with no report_id but status=failed → terminal.
			// Detected via stocks_status require_fields on failed stocks below.
		}
	}
	if !anyFail {
		report.Verdict = VerdictPass
		return report
	}
	// Classify: a stock in "failed" status is terminal; missing fields are recoverable.
	for code, ws := range working.Stocks {
		if ws.Status == "failed" {
			hardFail = true
			report.MissingSteps = append(report.MissingSteps, fmt.Sprintf("%s: stock status failed", code))
		}
	}
	if hardFail {
		report.Verdict = VerdictTerminal
	} else {
		report.Verdict = VerdictRecoverable
		// List missing per-stock steps as recoverable targets.
		for code, ws := range working.Stocks {
			if ws.Status != "reported" && ws.Status != "skipped" {
				report.MissingSteps = append(report.MissingSteps, fmt.Sprintf("%s: status=%s", code, ws.Status))
			} else if ws.Status == "reported" && ws.ReportID == "" {
				report.MissingSteps = append(report.MissingSteps, fmt.Sprintf("%s: missing report_id", code))
			}
		}
	}
	return report
}

func (e *Engine) runCheck(c Check, working *memory.PreMarketWorking, date string) CheckResult {
	res := CheckResult{Name: c.Name, Type: c.Type}
	switch c.Type {
	case "stocks_status":
		if c.ExpectPhase != "" {
			if working.Phase != c.ExpectPhase {
				res.Passed = false
				res.Detail = fmt.Sprintf("phase=%s, want %s", working.Phase, c.ExpectPhase)
				return res
			}
			// phase matches; if no stocks (non-trading day), pass.
			if len(working.Stocks) == 0 {
				res.Passed = true
				res.Detail = "no stocks (non-trading day)"
				return res
			}
		}
		// Check require_fields on stocks with for_status.
		missing := []string{}
		for code, ws := range working.Stocks {
			if c.ForStatus != "" && ws.Status != c.ForStatus {
				continue
			}
			for _, f := range c.RequireFields {
				if !stockHasField(ws, f) {
					missing = append(missing, fmt.Sprintf("%s.%s", code, f))
				}
			}
		}
		if len(missing) == 0 {
			res.Passed = true
			res.Detail = "all required fields present"
		} else {
			res.Passed = false
			res.Detail = "missing: " + strings.Join(missing, ", ")
		}
	case "file_exists":
		missingFiles := []string{}
		for code, ws := range working.Stocks {
			if c.ForStatus != "" && ws.Status != c.ForStatus {
				continue
			}
			pattern := c.Pattern
			pattern = strings.ReplaceAll(pattern, "{date}", date)
			pattern = strings.ReplaceAll(pattern, "{code}", code)
			full := filepath.Join(e.workspaceRoot, pattern)
			if _, err := os.Stat(full); err != nil {
				missingFiles = append(missingFiles, full)
			}
		}
		if len(missingFiles) == 0 {
			res.Passed = true
			res.Detail = "all local md present"
		} else {
			res.Passed = false
			res.Detail = "missing: " + strings.Join(missingFiles, ", ")
		}
	case "api_response":
		// Live API probe deferred to P6 (tool contract). Mark as skipped so it
		// does not influence verdict until a real probe is wired.
		res.Skipped = true
		res.Detail = "api_response check not yet wired (P6)"
	default:
		res.Skipped = true
		res.Detail = "unknown check type: " + c.Type
	}
	return res
}

func stockHasField(ws memory.StockWorkspace, field string) bool {
	switch field {
	case "report_id":
		return ws.ReportID != ""
	case "bot_id":
		return ws.BotID != ""
	case "bot_name":
		return ws.BotName != ""
	case "bot_type":
		return ws.BotType != ""
	case "report_ref":
		return ws.ReportRef != ""
	default:
		return false
	}
}
