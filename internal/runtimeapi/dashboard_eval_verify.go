package runtimeapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/eval"
)

func (h *Handler) evalJudge() eval.ReplyJudge {
	if h == nil || h.App == nil {
		return nil
	}
	provider := h.App.OpsBackgroundProvider()
	if provider == nil {
		return nil
	}
	return &eval.LLMReplyJudge{Provider: provider, Policy: h.App.OpsBackgroundPolicy()}
}

func (h *Handler) persistEvalVerifyRun(ctx context.Context, db *sql.DB, userID, runID, caseID, title, sessionID string, result eval.LiveVerifyResult, durationMs int64) error {
	if db == nil || runID == "" {
		return nil
	}
	_ = h.ensureEvalRunChecksTable(ctx, db)

	status := "pass"
	if !result.Passed {
		status = "fail"
	}
	summaryJSON, _ := json.Marshal(result.Summary)
	dialogueJSON, _ := json.Marshal(result.DialogueSnapshot)
	now := time.Now().UTC()
	started := now.Add(-time.Duration(durationMs) * time.Millisecond)

	_, err := db.ExecContext(ctx, h.evalSQL(`
		INSERT INTO agent_eval_runs (
			id, user_id, case_id, title, status, dual_model, model_slot_a, model_slot_b,
			duration_ms, error_text, logs_json, dialogue_snapshot_json, summary_json,
			started_at, ended_at, created_at
		) VALUES (?, ?, ?, ?, ?, 0, '', '', ?, ?, '[]', ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			duration_ms = excluded.duration_ms,
			error_text = excluded.error_text,
			dialogue_snapshot_json = excluded.dialogue_snapshot_json,
			summary_json = excluded.summary_json,
			ended_at = excluded.ended_at`),
		runID, userID, caseID, title, status, durationMs, detailIfFail(result),
		string(dialogueJSON), string(summaryJSON), started, now, now)
	if err != nil && !h.usesPostgresEval() {
		_, err = db.ExecContext(ctx, h.evalSQL(`
			INSERT OR REPLACE INTO agent_eval_runs (
				id, user_id, case_id, title, status, dual_model, model_slot_a, model_slot_b,
				duration_ms, error_text, logs_json, dialogue_snapshot_json, summary_json,
				started_at, ended_at, created_at
			) VALUES (?, ?, ?, ?, ?, 0, '', '', ?, ?, '[]', ?, ?, ?, ?, ?)`),
			runID, userID, caseID, title, status, durationMs, detailIfFail(result),
			string(dialogueJSON), string(summaryJSON), started.Format(time.RFC3339), now.Format(time.RFC3339), now.Format(time.RFC3339))
	}
	if err != nil {
		return err
	}

	for _, check := range result.Checks {
		checkID := runID + ":" + check.Type
		expectedJSON, _ := json.Marshal(check.Expected)
		actualJSON, _ := json.Marshal(check.Actual)
		var score sql.NullFloat64
		if check.Score != nil {
			score = sql.NullFloat64{Float64: *check.Score, Valid: true}
		}
		_, err = db.ExecContext(ctx, h.evalSQL(`
			INSERT INTO agent_eval_run_checks (
				id, run_id, case_id, session_id, check_type, passed, score,
				expected_json, actual_json, detail, model, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				passed = excluded.passed,
				score = excluded.score,
				expected_json = excluded.expected_json,
				actual_json = excluded.actual_json,
				detail = excluded.detail,
				model = excluded.model`),
			checkID, runID, caseID, sessionID, check.Type, check.Passed, score,
			string(expectedJSON), string(actualJSON), check.Detail, check.Model, now)
		if err != nil && !h.usesPostgresEval() {
			_, err = db.ExecContext(ctx, h.evalSQL(`
				INSERT OR REPLACE INTO agent_eval_run_checks (
					id, run_id, case_id, session_id, check_type, passed, score,
					expected_json, actual_json, detail, model, created_at
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
				checkID, runID, caseID, sessionID, check.Type, check.Passed, score,
				string(expectedJSON), string(actualJSON), check.Detail, check.Model, now.Format(time.RFC3339))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func detailIfFail(result eval.LiveVerifyResult) string {
	if result.Passed {
		return ""
	}
	return result.Detail
}

func (h *Handler) ensureEvalRunChecksTable(ctx context.Context, db *sql.DB) error {
	if h.usesPostgresEval() {
		_, err := db.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS agent_eval_run_checks (
				id              TEXT PRIMARY KEY,
				run_id          TEXT NOT NULL,
				case_id         TEXT NOT NULL DEFAULT '',
				session_id      TEXT NOT NULL DEFAULT '',
				check_type      TEXT NOT NULL,
				passed          BOOLEAN NOT NULL,
				score           REAL,
				expected_json   TEXT NOT NULL DEFAULT '{}',
				actual_json     TEXT NOT NULL DEFAULT '{}',
				detail          TEXT NOT NULL DEFAULT '',
				model           TEXT NOT NULL DEFAULT '',
				created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);
			CREATE INDEX IF NOT EXISTS idx_agent_eval_run_checks_run
				ON agent_eval_run_checks (run_id, check_type);
			ALTER TABLE agent_eval_runs ADD COLUMN IF NOT EXISTS dialogue_snapshot_json TEXT NOT NULL DEFAULT '[]';
			ALTER TABLE agent_eval_runs ADD COLUMN IF NOT EXISTS summary_json TEXT NOT NULL DEFAULT '{}';`)
		return err
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS agent_eval_run_checks (
			id TEXT PRIMARY KEY,
			run_id TEXT NOT NULL,
			case_id TEXT NOT NULL DEFAULT '',
			session_id TEXT NOT NULL DEFAULT '',
			check_type TEXT NOT NULL,
			passed INTEGER NOT NULL,
			score REAL,
			expected_json TEXT NOT NULL DEFAULT '{}',
			actual_json TEXT NOT NULL DEFAULT '{}',
			detail TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_agent_eval_run_checks_run ON agent_eval_run_checks (run_id, check_type);`,
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	// Best-effort column adds for existing SQLite DBs.
	for _, col := range []string{
		`ALTER TABLE agent_eval_runs ADD COLUMN dialogue_snapshot_json TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE agent_eval_runs ADD COLUMN summary_json TEXT NOT NULL DEFAULT '{}'`,
	} {
		_, _ = db.ExecContext(ctx, col)
	}
	return nil
}
