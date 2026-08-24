package app

import (
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/jobstore"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func (a *App) syncScheduledJobVerdict(skill, market string, result workflow.RunResult, runErr error) {
	if a == nil || strings.TrimSpace(a.Workspace) == "" {
		return
	}
	verdict := schedulerVerdictFromResult(result, runErr)
	if verdict == "" {
		return
	}
	dir := filepath.Join(a.Workspace, "scheduler")
	if err := jobstore.RecordSkillVerdict(dir, skill, market, verdict); err != nil {
		slog.Warn("app: sync scheduler job verdict failed", "skill", skill, "market", market, "verdict", verdict, "err", err)
	}
}

func schedulerVerdictFromResult(result workflow.RunResult, runErr error) string {
	if runErr != nil {
		return "error"
	}
	if result.Supervisor != nil {
		return string(result.Supervisor.Verdict)
	}
	if result.OK() {
		return "pass"
	}
	if strings.TrimSpace(result.LastError) != "" {
		return "error"
	}
	return ""
}
