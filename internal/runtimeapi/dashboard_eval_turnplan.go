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

var (
	errEvalCaseNotFound    = fmt.Errorf("case not found")
	errEvalCaseNotTurnPlan = fmt.Errorf("case is not turn_plan eval")
)

func (h *Handler) registerEvalTurnPlanRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/dashboard/eval/run-turn-plan", h.evalRunTurnPlan)
	mux.HandleFunc("POST /v1/dashboard/eval/cases/{id}/run", h.evalCaseRun)
	mux.HandleFunc("POST /v1/dashboard/eval/cases/{id}/verify", h.evalCaseVerify)
}

type evalRunTurnPlanRequest struct {
	CaseID string `json:"case_id"`
}

type evalCaseVerifyRequest struct {
	SessionID string `json:"session_id"`
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
	suite := eval.DefaultTurnPlanSuite()
	report := eval.RunTurnPlanReport(suite)
	writeJSON(w, map[string]any{
		"ok":          report.AllPass,
		"plan_only":   true,
		"passed":      report.Passed,
		"failed":      report.Failed,
		"total":       report.Total,
		"duration_ms": report.DurationMs,
		"results":     report.Results,
	})
}

func (h *Handler) evalCaseRun(w http.ResponseWriter, r *http.Request) {
	caseID := strings.TrimSpace(r.PathValue("id"))
	opts, title, err := h.loadTurnPlanCaseOptions(r, caseID)
	if err != nil {
		if err == errEvalCaseNotFound {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if opts.PlanOnly {
		suite := eval.TurnPlanSuite{
			Category: "turn_plan",
			PlanOnly: true,
			Turns:    []eval.TurnPlanTurn{{ID: opts.TurnID, Message: opts.Message, ExpectDomain: opts.ExpectDomain, ExpectMode: opts.ExpectMode, ExpectSOP: opts.ExpectSOP, ForbidTools: opts.ForbidTools, RequireTools: opts.RequireTools}},
		}
		report := eval.RunTurnPlanReport(suite)
		writeJSON(w, map[string]any{"ok": report.AllPass, "plan_only": true, "title": title, "results": report.Results})
		return
	}
	writeError(w, http.StatusBadRequest, "turn_plan live eval must run through Dock Chat; call POST /v1/dashboard/eval/cases/"+caseID+"/verify after the chat turn completes")
}

func (h *Handler) evalCaseVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	caseID := strings.TrimSpace(r.PathValue("id"))
	opts, title, err := h.loadTurnPlanCaseOptions(r, caseID)
	if err != nil {
		if err == errEvalCaseNotFound {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if opts.PlanOnly {
		writeError(w, http.StatusBadRequest, "case is plan_only; use POST /v1/dashboard/eval/run-turn-plan")
		return
	}

	var req evalCaseVerifyRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session_id required")
		return
	}

	store, err := h.App.SessionStore()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	chat, err := store.Load(sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if chat == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	start := time.Now()
	result := eval.VerifyTurnPlanLive(chat, opts)
	status := "pass"
	if !result.Passed {
		status = "fail"
	}

	writeJSON(w, map[string]any{
		"ok":          result.Passed,
		"case_id":     caseID,
		"title":       title,
		"status":      status,
		"plan_only":   false,
		"session_id":  sessionID,
		"turn_id":     result.TurnID,
		"detail":      result.Detail,
		"duration_ms": time.Since(start).Milliseconds(),
	})
}

func (h *Handler) loadTurnPlanCaseOptions(r *http.Request, caseID string) (eval.TurnPlanCaseOptions, string, error) {
	if caseID == "" {
		return eval.TurnPlanCaseOptions{}, "", fmt.Errorf("case id required")
	}
	for _, def := range eval.IndividualTurnPlanEvalCases() {
		if def.ID == caseID {
			return def.Options, def.Title, nil
		}
	}
	db := h.dashboardSQLDB()
	if db == nil {
		return eval.TurnPlanCaseOptions{}, "", errEvalCaseNotFound
	}
	userID := resolveUserID(r)
	row := db.QueryRowContext(r.Context(), h.evalSQL(`
		SELECT title, options_json
		FROM agent_eval_cases
		WHERE id = ? AND enabled = TRUE AND (user_id = '' OR user_id = ?)`), caseID, userID)
	var title, optsJSON string
	if err := row.Scan(&title, &optsJSON); err != nil {
		return eval.TurnPlanCaseOptions{}, "", errEvalCaseNotFound
	}
	opts, err := eval.ParseTurnPlanCaseOptions([]byte(optsJSON))
	if err != nil {
		return eval.TurnPlanCaseOptions{}, "", err
	}
	if opts.Category != "turn_plan" {
		return eval.TurnPlanCaseOptions{}, "", errEvalCaseNotTurnPlan
	}
	return opts, title, nil
}

func enrichEvalCaseRow(row map[string]any) {
	opts, _ := row["options"].(map[string]any)
	if opts == nil {
		return
	}
	category, _ := opts["category"].(string)
	if category != "turn_plan" {
		return
	}
	planOnly, _ := opts["plan_only"].(bool)
	if planOnly {
		row["run_mode"] = "plan_only"
		return
	}
	row["run_mode"] = "turn_plan_live"
	if msg, ok := opts["message"].(string); ok && strings.TrimSpace(msg) != "" {
		row["utterance"] = strings.TrimSpace(msg)
	}
	if setup, ok := opts["setup_messages"].([]any); ok && len(setup) > 0 {
		out := make([]string, 0, len(setup))
		for _, item := range setup {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		if len(out) > 0 {
			row["setup_utterances"] = out
		}
	}
}

func newEvalRunID(caseID string) string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("run-%s-%d-%s", caseID, time.Now().Unix(), hex.EncodeToString(b[:]))
}
