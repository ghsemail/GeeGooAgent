package runtimeapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/eval"
)

func (h *Handler) registerEvalTurnPlanRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/dashboard/eval/run-turn-plan", h.evalRunTurnPlan)
	mux.HandleFunc("POST /v1/dashboard/eval/cases/{id}/run", h.evalCaseRun)
}

type evalRunTurnPlanRequest struct {
	CaseID string `json:"case_id"`
}

func (h *Handler) evalRunTurnPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req evalRunTurnPlanRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	caseID := strings.TrimSpace(req.CaseID)
	if caseID == "" {
		caseID = "turn_plan_routing"
	}
	h.runTurnPlanEval(w, r, caseID)
}

func (h *Handler) evalCaseRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	caseID := strings.TrimSpace(r.PathValue("id"))
	if caseID == "" {
		writeError(w, http.StatusBadRequest, "case id required")
		return
	}
	h.runTurnPlanEval(w, r, caseID)
}

func (h *Handler) runTurnPlanEval(w http.ResponseWriter, r *http.Request, caseID string) {
	suite, title, err := h.loadTurnPlanSuite(r, caseID)
	if err != nil {
		if err == errEvalStorageUnavailable {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		if err == errEvalCaseNotFound {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if err == errEvalCaseNotTurnPlan {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	report := eval.RunTurnPlanReport(suite)
	report.CaseID = caseID
	runID := newEvalRunID(caseID)
	status := "pass"
	if !report.AllPass {
		status = "fail"
	}

	logs := turnPlanRunLogs(report)
	if db := h.dashboardSQLDB(); db != nil {
		userID := resolveUserID(r)
		now := time.Now().UTC()
		duration := int(report.DurationMs)
		logsJSON, _ := json.Marshal(logs)
		_, dbErr := db.ExecContext(r.Context(), h.evalSQL(`
			INSERT INTO agent_eval_runs (
				id, user_id, case_id, title, status, dual_model, model_slot_a, model_slot_b,
				duration_ms, error_text, logs_json, started_at, ended_at, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				case_id = excluded.case_id,
				title = excluded.title,
				status = excluded.status,
				duration_ms = excluded.duration_ms,
				error_text = excluded.error_text,
				logs_json = excluded.logs_json,
				started_at = excluded.started_at,
				ended_at = excluded.ended_at`),
			runID, userID, caseID, title, status, false, "", "",
			duration, turnPlanRunError(report), string(logsJSON), now, now, now)
		if dbErr != nil && !h.usesPostgresEval() {
			_, dbErr2 := db.ExecContext(r.Context(), h.evalSQL(`
				INSERT OR REPLACE INTO agent_eval_runs (
					id, user_id, case_id, title, status, dual_model, model_slot_a, model_slot_b,
					duration_ms, error_text, logs_json, started_at, ended_at, created_at
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
				runID, userID, caseID, title, status, false, "", "",
				duration, turnPlanRunError(report), string(logsJSON), now.Format(time.RFC3339),
				now.Format(time.RFC3339), now.Format(time.RFC3339))
			if dbErr2 != nil {
				writeError(w, http.StatusInternalServerError, dbErr.Error())
				return
			}
		} else if dbErr != nil {
			writeError(w, http.StatusInternalServerError, dbErr.Error())
			return
		}
	}

	writeJSON(w, map[string]any{
		"ok":          report.AllPass,
		"run_id":      runID,
		"case_id":     caseID,
		"title":       title,
		"status":      status,
		"plan_only":   true,
		"passed":      report.Passed,
		"failed":      report.Failed,
		"total":       report.Total,
		"duration_ms": report.DurationMs,
		"results":     report.Results,
	})
}

var (
	errEvalStorageUnavailable = fmt.Errorf("eval storage not configured")
	errEvalCaseNotFound       = fmt.Errorf("case not found")
	errEvalCaseNotTurnPlan    = fmt.Errorf("case is not turn_plan plan_only eval")
)

func (h *Handler) loadTurnPlanSuite(r *http.Request, caseID string) (eval.TurnPlanSuite, string, error) {
	if caseID == "builtin" || caseID == "default" {
		return eval.DefaultTurnPlanSuite(), "TurnPlan · 意图路由回归", nil
	}
	db := h.dashboardSQLDB()
	if db == nil {
		if caseID == "turn_plan_routing" || caseID == "builtin" || caseID == "default" {
			return eval.DefaultTurnPlanSuite(), "TurnPlan · 意图路由回归", nil
		}
		return eval.TurnPlanSuite{}, "", errEvalCaseNotFound
	}
	userID := resolveUserID(r)
	row := db.QueryRowContext(r.Context(), h.evalSQL(`
		SELECT title, options_json
		FROM agent_eval_cases
		WHERE id = ? AND enabled = TRUE AND (user_id = '' OR user_id = ?)`), caseID, userID)
	var title, optsJSON string
	if err := row.Scan(&title, &optsJSON); err != nil {
		if caseID == "turn_plan_routing" {
			return eval.DefaultTurnPlanSuite(), "TurnPlan · 意图路由回归", nil
		}
		return eval.TurnPlanSuite{}, "", errEvalCaseNotFound
	}
	suite, err := eval.ParseTurnPlanSuite([]byte(optsJSON))
	if err != nil {
		return eval.TurnPlanSuite{}, "", err
	}
	if suite.Category != "turn_plan" && !suite.PlanOnly {
		return eval.TurnPlanSuite{}, "", errEvalCaseNotTurnPlan
	}
	if len(suite.Turns) == 0 {
		return eval.TurnPlanSuite{}, "", fmt.Errorf("turn_plan suite has no turns")
	}
	return suite, title, nil
}

func newEvalRunID(caseID string) string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("run-%s-%d-%s", caseID, time.Now().Unix(), hex.EncodeToString(b[:]))
}

func turnPlanRunLogs(report eval.TurnPlanRunReport) []map[string]any {
	logs := make([]map[string]any, 0, len(report.Results)+1)
	logs = append(logs, map[string]any{
		"level":   "info",
		"message": fmt.Sprintf("TurnPlan eval: %d/%d passed in %dms", report.Passed, report.Total, report.DurationMs),
	})
	for _, res := range report.Results {
		level := "info"
		if !res.Passed {
			level = "error"
		}
		logs = append(logs, map[string]any{
			"level":   level,
			"turn_id": res.TurnID,
			"message": res.Message,
			"detail":  res.Detail,
			"passed":  res.Passed,
		})
	}
	return logs
}

func turnPlanRunError(report eval.TurnPlanRunReport) string {
	if report.AllPass {
		return ""
	}
	for _, res := range report.Results {
		if !res.Passed {
			return res.TurnID + ": " + res.Detail
		}
	}
	return "turn_plan eval failed"
}
