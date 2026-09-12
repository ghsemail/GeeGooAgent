package runtimeapi

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func (h *Handler) registerEvalRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/dashboard/eval/cases", h.evalCasesList)
	mux.HandleFunc("GET /v1/dashboard/eval/cases/{id}", h.evalCaseGet)
	mux.HandleFunc("POST /v1/dashboard/eval/cases", h.evalCaseCreate)
	mux.HandleFunc("PUT /v1/dashboard/eval/cases/{id}", h.evalCaseUpdate)
	mux.HandleFunc("DELETE /v1/dashboard/eval/cases/{id}", h.evalCaseDelete)

	mux.HandleFunc("GET /v1/dashboard/eval/runs", h.evalRunsList)
	mux.HandleFunc("GET /v1/dashboard/eval/runs/{id}", h.evalRunGet)
	mux.HandleFunc("POST /v1/dashboard/eval/runs", h.evalRunUpsert)
	mux.HandleFunc("DELETE /v1/dashboard/eval/runs/all", h.evalRunsClearAll)
	mux.HandleFunc("DELETE /v1/dashboard/eval/runs/{id}", h.evalRunDelete)
	h.registerEvalTurnPlanRoutes(mux)
	h.registerEvalAutoRoutes(mux)
	h.registerEvalSuggestRoutes(mux)
	h.registerEvalAutoClarifyRoutes(mux)
}

type evalCasePayload struct {
	ID                  string         `json:"id"`
	Title               string         `json:"title"`
	Description         string         `json:"description"`
	Steps               []string       `json:"steps"`
	SupportsRandomStock bool           `json:"supports_random_stock"`
	Options             map[string]any `json:"options"`
	SortOrder           int            `json:"sort_order"`
	Enabled             *bool          `json:"enabled"`
}

type evalRunPayload struct {
	ID         string           `json:"id"`
	CaseID     string           `json:"case_id"`
	Title      string           `json:"title"`
	Status     string           `json:"status"`
	DualModel  bool             `json:"dual_model"`
	ModelSlotA string           `json:"model_slot_a"`
	ModelSlotB string           `json:"model_slot_b"`
	SessionID  string           `json:"session_id"`
	Sessions   []map[string]any `json:"sessions"`
	DurationMs *int             `json:"duration_ms"`
	Error      string           `json:"error"`
	Logs       []map[string]any `json:"logs"`
	StartedAt  string           `json:"started_at"`
	EndedAt    string           `json:"ended_at"`
}

func (h *Handler) evalDBOrError(w http.ResponseWriter) *sql.DB {
	db := h.dashboardSQLDB()
	if db == nil {
		writeError(w, http.StatusServiceUnavailable, "eval storage not configured")
		return nil
	}
	return db
}

func (h *Handler) evalCasesList(w http.ResponseWriter, r *http.Request) {
	db := h.evalDBOrError(w)
	if db == nil {
		return
	}
	userID := resolveUserID(r)
	rows, err := db.QueryContext(r.Context(), h.evalSQL(`
		SELECT id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled
		FROM agent_eval_cases
		WHERE enabled = TRUE AND (user_id = '' OR user_id = ?)
		ORDER BY sort_order ASC, id ASC`), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		row, err := scanEvalCaseRow(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, row)
	}
	for i := range out {
		enrichEvalCaseRow(out[i])
	}
	writeJSON(w, map[string]any{"cases": out, "total": len(out)})
}

func (h *Handler) evalCaseGet(w http.ResponseWriter, r *http.Request) {
	db := h.evalDBOrError(w)
	if db == nil {
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "case id required")
		return
	}
	userID := resolveUserID(r)
	row := db.QueryRowContext(r.Context(), h.evalSQL(`
		SELECT id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled
		FROM agent_eval_cases
		WHERE id = ? AND enabled = TRUE AND (user_id = '' OR user_id = ?)`), id, userID)
	caseRow, err := scanEvalCaseRow(row)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "case not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	enrichEvalCaseRow(caseRow)
	writeJSON(w, map[string]any{"case": caseRow})
}

func (h *Handler) evalCaseCreate(w http.ResponseWriter, r *http.Request) {
	db := h.evalDBOrError(w)
	if db == nil {
		return
	}
	var req evalCasePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	id := strings.TrimSpace(req.ID)
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}
	userID := resolveUserID(r)
	now := time.Now().UTC()
	stepsJSON, _ := json.Marshal(req.Steps)
	optsJSON, _ := json.Marshal(persistEvalCaseOptions(req.Options))
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	_, err := db.ExecContext(r.Context(), h.evalSQL(`
		INSERT INTO agent_eval_cases (
			id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		id, userID, strings.TrimSpace(req.Title), strings.TrimSpace(req.Description),
		string(stepsJSON), req.SupportsRandomStock, string(optsJSON), req.SortOrder, enabled, now, now)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "id": id})
}

func (h *Handler) evalCaseUpdate(w http.ResponseWriter, r *http.Request) {
	db := h.evalDBOrError(w)
	if db == nil {
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "case id required")
		return
	}
	var req evalCasePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	userID := resolveUserID(r)
	stepsJSON, _ := json.Marshal(req.Steps)
	optsJSON, _ := json.Marshal(persistEvalCaseOptions(req.Options))
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	res, err := db.ExecContext(r.Context(), h.evalSQL(`
		UPDATE agent_eval_cases SET
			title = ?, description = ?, steps_json = ?, supports_random_stock = ?,
			options_json = ?, sort_order = ?, enabled = ?, updated_at = ?
		WHERE id = ? AND (user_id = '' OR user_id = ?)`),
		strings.TrimSpace(req.Title), strings.TrimSpace(req.Description), string(stepsJSON),
		req.SupportsRandomStock, string(optsJSON), req.SortOrder, enabled, time.Now().UTC(),
		id, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, http.StatusNotFound, "case not found")
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (h *Handler) evalCaseDelete(w http.ResponseWriter, r *http.Request) {
	db := h.evalDBOrError(w)
	if db == nil {
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "case id required")
		return
	}
	userID := resolveUserID(r)
	if id == "hello_then_stock_analysis" && userID == "" {
		writeError(w, http.StatusForbidden, "cannot delete built-in case")
		return
	}
	res, err := db.ExecContext(r.Context(), h.evalSQL(`
		DELETE FROM agent_eval_cases WHERE id = ? AND user_id = ?`), id, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, http.StatusNotFound, "case not found")
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (h *Handler) evalRunsList(w http.ResponseWriter, r *http.Request) {
	db := h.evalDBOrError(w)
	if db == nil {
		return
	}
	userID := resolveUserID(r)
	limit := parseLimit(r, 500, 1000)
	where, whereArgs := evalRunsWhereClause(userID)
	countArgs := append([]any{}, whereArgs...)
	var total int
	countSQL := h.evalSQL(`SELECT COUNT(*) FROM agent_eval_runs WHERE ` + where)
	if err := db.QueryRowContext(r.Context(), countSQL, countArgs...).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	listArgs := append(append([]any{}, whereArgs...), limit)
	rows, err := db.QueryContext(r.Context(), h.evalSQL(`
		SELECT id, case_id, title, status, dual_model, model_slot_a, model_slot_b,
		       session_id, sessions_json, duration_ms, error_text, logs_json, started_at, ended_at
		FROM agent_eval_runs
		WHERE `+where+`
		ORDER BY started_at DESC
		LIMIT ?`), listArgs...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		row, err := scanEvalRunRow(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, row)
	}
	writeJSON(w, map[string]any{
		"runs":      out,
		"total":     total,
		"truncated": len(out) < total,
	})
}

func (h *Handler) evalRunGet(w http.ResponseWriter, r *http.Request) {
	db := h.evalDBOrError(w)
	if db == nil {
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	userID := resolveUserID(r)
	where, whereArgs := evalRunsWhereClause(userID)
	getArgs := append([]any{id}, whereArgs...)
	row := db.QueryRowContext(r.Context(), h.evalSQL(`
		SELECT id, case_id, title, status, dual_model, model_slot_a, model_slot_b,
		       session_id, sessions_json, duration_ms, error_text, logs_json, started_at, ended_at
		FROM agent_eval_runs WHERE id = ? AND (`+where+`)`), getArgs...)
	runRow, err := scanEvalRunRow(row)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]any{"run": runRow})
}

func (h *Handler) evalRunUpsert(w http.ResponseWriter, r *http.Request) {
	db := h.evalDBOrError(w)
	if db == nil {
		return
	}
	var req evalRunPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	id := strings.TrimSpace(req.ID)
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}
	userID := resolveUserID(r)
	started, err := time.Parse(time.RFC3339, strings.TrimSpace(req.StartedAt))
	if err != nil {
		started = time.Now().UTC()
	}
	var ended sql.NullTime
	if strings.TrimSpace(req.EndedAt) != "" {
		if t, err := time.Parse(time.RFC3339, req.EndedAt); err == nil {
			ended = sql.NullTime{Time: t, Valid: true}
		}
	}
	logsJSON, _ := json.Marshal(req.Logs)
	sessionsJSON, _ := json.Marshal(req.Sessions)
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" && len(req.Sessions) > 0 {
		if sid, _ := req.Sessions[0]["session_id"].(string); strings.TrimSpace(sid) != "" {
			sessionID = strings.TrimSpace(sid)
		}
	}
	var duration sql.NullInt64
	if req.DurationMs != nil {
		duration = sql.NullInt64{Int64: int64(*req.DurationMs), Valid: true}
	}
	now := time.Now().UTC()
	_, err = db.ExecContext(r.Context(), h.evalSQL(`
		INSERT INTO agent_eval_runs (
			id, user_id, case_id, title, status, dual_model, model_slot_a, model_slot_b,
			session_id, sessions_json, duration_ms, error_text, logs_json, started_at, ended_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			case_id = excluded.case_id,
			title = excluded.title,
			status = excluded.status,
			dual_model = excluded.dual_model,
			model_slot_a = excluded.model_slot_a,
			model_slot_b = excluded.model_slot_b,
			session_id = excluded.session_id,
			sessions_json = excluded.sessions_json,
			duration_ms = excluded.duration_ms,
			error_text = excluded.error_text,
			logs_json = excluded.logs_json,
			started_at = excluded.started_at,
			ended_at = excluded.ended_at`),
		id, userID, strings.TrimSpace(req.CaseID), strings.TrimSpace(req.Title), strings.TrimSpace(req.Status),
		req.DualModel, strings.TrimSpace(req.ModelSlotA), strings.TrimSpace(req.ModelSlotB),
		sessionID, string(sessionsJSON), duration, strings.TrimSpace(req.Error), string(logsJSON), started, ended, now)
	if err != nil && !h.usesPostgresEval() {
		_, err2 := db.ExecContext(r.Context(), h.evalSQL(`
			INSERT OR REPLACE INTO agent_eval_runs (
				id, user_id, case_id, title, status, dual_model, model_slot_a, model_slot_b,
				session_id, sessions_json, duration_ms, error_text, logs_json, started_at, ended_at, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
			id, userID, strings.TrimSpace(req.CaseID), strings.TrimSpace(req.Title), strings.TrimSpace(req.Status),
			req.DualModel, strings.TrimSpace(req.ModelSlotA), strings.TrimSpace(req.ModelSlotB),
			sessionID, string(sessionsJSON), duration, strings.TrimSpace(req.Error), string(logsJSON), started.Format(time.RFC3339),
			endedTimeString(ended), now.Format(time.RFC3339))
		if err2 != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "id": id})
}

func endedTimeString(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.UTC().Format(time.RFC3339)
}

func (h *Handler) evalRunDelete(w http.ResponseWriter, r *http.Request) {
	db := h.evalDBOrError(w)
	if db == nil {
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" || id == "all" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}
	_ = h.ensureEvalRunChecksTable(r.Context(), db)
	_, _ = db.ExecContext(r.Context(), h.evalSQL(`DELETE FROM agent_eval_run_checks WHERE run_id = ?`), id)
	res, err := db.ExecContext(r.Context(), h.evalSQL(`DELETE FROM agent_eval_runs WHERE id = ?`), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (h *Handler) evalRunsClearAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	db := h.evalDBOrError(w)
	if db == nil {
		return
	}
	_ = h.ensureEvalRunChecksTable(r.Context(), db)
	_, _ = db.ExecContext(r.Context(), h.evalSQL(`DELETE FROM agent_eval_run_checks`))
	res, err := db.ExecContext(r.Context(), h.evalSQL(`DELETE FROM agent_eval_runs`))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	n, _ := res.RowsAffected()
	writeJSON(w, map[string]any{"ok": true, "deleted": n})
}

type evalCaseScanner interface {
	Scan(dest ...any) error
}

func scanEvalCaseRow(s evalCaseScanner) (map[string]any, error) {
	var id, title, description, stepsJSON, optsJSON string
	var supports bool
	var sortOrder int
	var enabled bool
	if err := s.Scan(&id, &title, &description, &stepsJSON, &supports, &optsJSON, &sortOrder, &enabled); err != nil {
		return nil, err
	}
	steps := []string{}
	_ = json.Unmarshal([]byte(stepsJSON), &steps)
	opts := map[string]any{}
	_ = json.Unmarshal([]byte(optsJSON), &opts)
	return map[string]any{
		"id":                    id,
		"title":                 title,
		"description":           description,
		"steps":                 steps,
		"supports_random_stock": supports,
		"options":               opts,
		"sort_order":            sortOrder,
		"enabled":               enabled,
	}, nil
}

func scanEvalRunRow(s evalCaseScanner) (map[string]any, error) {
	var id, caseID, title, status, modelA, modelB, sessionID, sessionsJSON, errText, logsJSON string
	var dual bool
	var duration sql.NullInt64
	var started, ended sql.NullTime
	if err := s.Scan(&id, &caseID, &title, &status, &dual, &modelA, &modelB, &sessionID, &sessionsJSON, &duration, &errText, &logsJSON, &started, &ended); err != nil {
		return nil, err
	}
	logs := []any{}
	_ = json.Unmarshal([]byte(logsJSON), &logs)
	sessions := []any{}
	_ = json.Unmarshal([]byte(sessionsJSON), &sessions)
	out := map[string]any{
		"id":           id,
		"case_id":      caseID,
		"title":        title,
		"status":       normalizeEvalRunStatus(status),
		"dual_model":   dual,
		"model_slot_a": modelA,
		"model_slot_b": modelB,
		"session_id":   sessionID,
		"sessions":     sessions,
		"error":        errText,
		"logs":         logs,
	}
	if duration.Valid {
		out["duration_ms"] = duration.Int64
	}
	if started.Valid {
		out["started_at"] = started.Time.UTC().Format(time.RFC3339)
	}
	if ended.Valid {
		out["ended_at"] = ended.Time.UTC().Format(time.RFC3339)
	}
	return out, nil
}
