package app

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/stockdigest"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

// SkillDeliveryReasonForScheduler is the scheduler retry boundary: the skill run
// must complete with pass verdict (and stock jobs must return no delivery error).
func SkillDeliveryReasonForScheduler(result workflow.RunResult, runErr error) (bool, string) {
	if runErr != nil {
		return false, runErr.Error()
	}
	if !workflowDeliveryPass(result) {
		if result.Supervisor != nil && result.Supervisor.Verdict != workflow.VerdictPass {
			return false, "supervisor_" + string(result.Supervisor.Verdict)
		}
		if strings.TrimSpace(result.LastError) != "" {
			return false, result.LastError
		}
		return false, "workflow_not_completed"
	}
	return true, ""
}

// userStockDeliveryOK reports whether one user's stock report was generated and,
// when notifyRequired is set, pushed to Feishu successfully.
func userStockDeliveryOK(
	skill, market string,
	result workflow.RunResult,
	runErr error,
	notifyRequired bool,
	feishuSent bool,
	feishuSkipReason string,
) (bool, string) {
	if runErr != nil {
		return false, "workflow_error"
	}
	if !result.OK() {
		return false, "workflow_not_completed"
	}
	if result.Supervisor != nil && result.Supervisor.Verdict != workflow.VerdictPass {
		if result.Supervisor.Verdict == workflow.VerdictRecoverable && stockdigest.HasNewlyReportedStock(result) {
			// Report already persisted; avoid re-running the whole job and duplicating rows.
		} else {
			return false, "supervisor_" + string(result.Supervisor.Verdict)
		}
	}
	skipReason := stockdigest.NotifySkipReason(skill, market, result)
	if skipReason == "non_trading_day" || skipReason == "no_new_reports" {
		// no_new_reports: today's report already exists (idempotent re-run), not a failure.
		return true, ""
	}
	if skipReason != "" {
		return false, skipReason
	}
	if notifyRequired {
		if feishuSent {
			return true, ""
		}
		if r := strings.TrimSpace(feishuSkipReason); r != "" {
			if r == "no_new_reports" {
				return true, ""
			}
			return false, r
		}
		return false, "feishu_not_sent"
	}
	return true, ""
}

func workflowDeliveryPass(result workflow.RunResult) bool {
	if result.Supervisor != nil {
		return result.Supervisor.Verdict == workflow.VerdictPass
	}
	return result.OK()
}
