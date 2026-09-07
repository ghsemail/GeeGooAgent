package runtimeapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/eval"
)

func (h *Handler) registerEvalAutoRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/dashboard/eval/catalog", h.evalCatalog)
	mux.HandleFunc("GET /v1/dashboard/eval/suites", h.evalSuitesList)
	mux.HandleFunc("POST /v1/dashboard/eval/suites", h.evalSuiteCreate)
	mux.HandleFunc("GET /v1/dashboard/eval/suites/{id}", h.evalSuiteGet)
	mux.HandleFunc("DELETE /v1/dashboard/eval/suites/{id}", h.evalSuiteDelete)
	mux.HandleFunc("GET /v1/dashboard/eval/jobs", h.evalJobsList)
	mux.HandleFunc("POST /v1/dashboard/eval/jobs", h.evalJobCreate)
	mux.HandleFunc("GET /v1/dashboard/eval/jobs/{id}", h.evalJobGet)
	mux.HandleFunc("POST /v1/dashboard/eval/jobs/{id}/cancel", h.evalJobCancel)
	mux.HandleFunc("GET /v1/dashboard/eval/jobs/{id}/items/{item_id}", h.evalJobItemDetail)
	mux.HandleFunc("GET /v1/dashboard/eval/auto", h.evalAutoPage)
}

type evalCatalogCase struct {
	ID           string   `json:"id"`
	TurnID       string   `json:"turn_id"`
	Category     string   `json:"category"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Setup        []string `json:"setup,omitempty"`
	Message      string   `json:"message"`
	ExpectDomain string   `json:"expect_domain,omitempty"`
	ExpectMode   string   `json:"expect_mode,omitempty"`
}

type evalCatalogCategory struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Order int    `json:"order"`
	Count int    `json:"count"`
}

type evalSuitePayload struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	CategoryIDs []string `json:"category_ids"`
	CaseIDs     []string `json:"case_ids"`
}

type evalJobCreatePayload struct {
	Title       string   `json:"title"`
	SuiteID     string   `json:"suite_id"`
	CategoryIDs []string `json:"category_ids"`
	CaseIDs     []string `json:"case_ids"`
	MCPToken    string   `json:"mcp_token"`
}

func evalCatalogCases() []evalCatalogCase {
	live := eval.DefaultTurnPlanLiveCases()
	byTurn := make(map[string]eval.TurnPlanLiveCase, len(live))
	for _, c := range live {
		byTurn[c.ID] = c
	}
	out := make([]evalCatalogCase, 0, len(live))
	for _, def := range eval.IndividualTurnPlanEvalCases() {
		src := byTurn[def.Options.TurnID]
		out = append(out, evalCatalogCase{
			ID:           def.ID,
			TurnID:       def.Options.TurnID,
			Category:     src.Category,
			Title:        def.Title,
			Description:  def.Description,
			Setup:        append([]string(nil), def.Options.SetupMessages...),
			Message:      def.Options.Message,
			ExpectDomain: def.Options.ExpectDomain,
			ExpectMode:   def.Options.ExpectMode,
		})
	}
	return out
}

func evalCatalogCategoryList(cases []evalCatalogCase) []evalCatalogCategory {
	counts := map[string]int{}
	for _, c := range cases {
		counts[c.Category]++
	}
	out := make([]evalCatalogCategory, 0, 8)
	for _, cat := range eval.TurnPlanCategories() {
		out = append(out, evalCatalogCategory{
			ID: cat.ID, Title: cat.Title, Order: cat.Order, Count: counts[cat.ID],
		})
	}
	return out
}

func expandEvalAutoCases(categoryIDs, caseIDs []string) ([]evalCatalogCase, error) {
	all := evalCatalogCases()
	byID := make(map[string]evalCatalogCase, len(all))
	for _, c := range all {
		byID[c.ID] = c
	}
	caseIDs = uniqueNonEmpty(caseIDs)
	categoryIDs = uniqueNonEmpty(categoryIDs)
	if len(caseIDs) > 0 {
		out := make([]evalCatalogCase, 0, len(caseIDs))
		for _, id := range caseIDs {
			c, ok := byID[id]
			if !ok {
				return nil, fmt.Errorf("unknown case: %s", id)
			}
			if len(categoryIDs) > 0 && !containsFold(categoryIDs, c.Category) {
				continue
			}
			out = append(out, c)
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("no cases matched the selected filters")
		}
		return out, nil
	}
	if len(categoryIDs) > 0 {
		out := make([]evalCatalogCase, 0)
		for _, c := range all {
			if containsFold(categoryIDs, c.Category) {
				out = append(out, c)
			}
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("no cases in selected categories")
		}
		return out, nil
	}
	return all, nil
}

func (h *Handler) evalCatalog(w http.ResponseWriter, r *http.Request) {
	cases := evalCatalogCases()
	writeJSON(w, map[string]any{
		"categories": evalCatalogCategoryList(cases),
		"cases":      cases,
		"total":      len(cases),
	})
}

func (h *Handler) evalSuiteCreate(w http.ResponseWriter, r *http.Request) {
	db := h.evalDBOrError(w)
	if db == nil {
		return
	}
	if err := h.ensureEvalAutoTables(r.Context(), db); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var req evalSuitePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		writeError(w, http.StatusBadRequest, "title required")
		return
	}
	picked, err := expandEvalAutoCases(req.CategoryIDs, req.CaseIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	caseIDs := make([]string, 0, len(picked))
	catSet := map[string]struct{}{}
	for _, c := range picked {
		caseIDs = append(caseIDs, c.ID)
		if c.Category != "" {
			catSet[c.Category] = struct{}{}
		}
	}
	categoryIDs := uniqueNonEmpty(req.CategoryIDs)
	if len(categoryIDs) == 0 {
		for id := range catSet {
			categoryIDs = append(categoryIDs, id)
		}
	}
	id := "suite-" + shortEvalID()
	now := time.Now().UTC().Format(time.RFC3339)
	userID := resolveUserID(r)
	catsJSON, _ := json.Marshal(categoryIDs)
	casesJSON, _ := json.Marshal(caseIDs)
	_, err = db.ExecContext(r.Context(), h.evalSQL(`
		INSERT INTO agent_eval_suites (id, user_id, title, description, category_ids_json, case_ids_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`),
		id, userID, title, strings.TrimSpace(req.Description), string(catsJSON), string(casesJSON), now, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "suite": h.loadEvalSuite(r.Context(), db, userID, id)})
}

func (h *Handler) evalSuitesList(w http.ResponseWriter, r *http.Request) {
	db := h.evalDBOrError(w)
	if db == nil {
		return
	}
	if err := h.ensureEvalAutoTables(r.Context(), db); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	userID := resolveUserID(r)
	rows, err := db.QueryContext(r.Context(), h.evalSQL(`
		SELECT id, title, description, category_ids_json, case_ids_json, created_at, updated_at
		FROM agent_eval_suites
		WHERE user_id = ? OR (? = '' AND user_id = '')
		ORDER BY updated_at DESC`), userID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		row, err := scanEvalSuiteRow(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, row)
	}
	writeJSON(w, map[string]any{"suites": out, "total": len(out)})
}

func (h *Handler) evalSuiteGet(w http.ResponseWriter, r *http.Request) {
	db := h.evalDBOrError(w)
	if db == nil {
		return
	}
	if err := h.ensureEvalAutoTables(r.Context(), db); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	suite := h.loadEvalSuite(r.Context(), db, resolveUserID(r), id)
	if suite == nil {
		writeError(w, http.StatusNotFound, "suite not found")
		return
	}
	writeJSON(w, map[string]any{"suite": suite})
}

func (h *Handler) evalSuiteDelete(w http.ResponseWriter, r *http.Request) {
	db := h.evalDBOrError(w)
	if db == nil {
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	userID := resolveUserID(r)
	res, err := db.ExecContext(r.Context(), h.evalSQL(`
		DELETE FROM agent_eval_suites WHERE id = ? AND (user_id = ? OR (? = '' AND user_id = ''))`),
		id, userID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, http.StatusNotFound, "suite not found")
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (h *Handler) loadEvalSuite(ctx context.Context, db *sql.DB, userID, id string) map[string]any {
	if db == nil || strings.TrimSpace(id) == "" {
		return nil
	}
	row := db.QueryRowContext(ctx, h.evalSQL(`
		SELECT id, title, description, category_ids_json, case_ids_json, created_at, updated_at
		FROM agent_eval_suites
		WHERE id = ? AND (user_id = ? OR (? = '' AND user_id = ''))`), id, userID, userID)
	out, err := scanEvalSuiteRow(row)
	if err != nil {
		return nil
	}
	return out
}

func scanEvalSuiteRow(s evalCaseScanner) (map[string]any, error) {
	var id, title, description, catsJSON, casesJSON, created, updated string
	if err := s.Scan(&id, &title, &description, &catsJSON, &casesJSON, &created, &updated); err != nil {
		return nil, err
	}
	return map[string]any{
		"id":           id,
		"title":        title,
		"description":  description,
		"category_ids": decodeStringSlice(catsJSON),
		"case_ids":     decodeStringSlice(casesJSON),
		"created_at":   created,
		"updated_at":   updated,
	}, nil
}

func (h *Handler) evalJobCreate(w http.ResponseWriter, r *http.Request) {
	db := h.evalDBOrError(w)
	if db == nil {
		return
	}
	if err := h.ensureEvalAutoTables(r.Context(), db); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var req evalJobCreatePayload
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	userID := resolveUserID(r)
	categoryIDs := uniqueNonEmpty(req.CategoryIDs)
	caseIDs := uniqueNonEmpty(req.CaseIDs)
	title := strings.TrimSpace(req.Title)
	suiteID := strings.TrimSpace(req.SuiteID)
	if suiteID != "" {
		suite := h.loadEvalSuite(r.Context(), db, userID, suiteID)
		if suite == nil {
			writeError(w, http.StatusNotFound, "suite not found")
			return
		}
		if title == "" {
			title, _ = suite["title"].(string)
		}
		if len(categoryIDs) == 0 {
			categoryIDs = asStringSlice(suite["category_ids"])
		}
		if len(caseIDs) == 0 {
			caseIDs = asStringSlice(suite["case_ids"])
		}
	}
	picked, err := expandEvalAutoCases(categoryIDs, caseIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if title == "" {
		title = "自动测评"
		if len(categoryIDs) == 1 {
			title = "自动测评 · " + categoryIDs[0]
		}
	}
	mcpToken := resolveChatMCPToken(r, req.MCPToken, h.configMCPToken())
	if mcpToken == "" {
		writeError(w, http.StatusUnauthorized, "missing mcp_token: set mcp_token in agent-runtime config")
		return
	}

	jobID := "job-" + shortEvalID()
	now := time.Now().UTC().Format(time.RFC3339)
	storedCaseIDs := make([]string, 0, len(picked))
	for _, c := range picked {
		storedCaseIDs = append(storedCaseIDs, c.ID)
	}
	catsJSON, _ := json.Marshal(categoryIDs)
	casesJSON, _ := json.Marshal(storedCaseIDs)
	_, err = db.ExecContext(r.Context(), h.evalSQL(`
		INSERT INTO agent_eval_jobs (
			id, user_id, suite_id, title, status, category_ids_json, case_ids_json,
			total, passed, failed, error_text, started_at, ended_at, created_at
		) VALUES (?, ?, ?, ?, 'queued', ?, ?, ?, 0, 0, '', ?, '', ?)`),
		jobID, userID, suiteID, title, string(catsJSON), string(casesJSON), len(picked), now, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, c := range picked {
		itemID := jobID + ":" + c.ID
		_, err = db.ExecContext(r.Context(), h.evalSQL(`
			INSERT INTO agent_eval_job_items (
				id, job_id, case_id, title, category, status, session_id, run_id,
				detail, summary_json, checks_json, loop_error, duration_ms, started_at, ended_at, created_at
			) VALUES (?, ?, ?, ?, ?, 'pending', '', '', '', '{}', '[]', '', 0, '', '', ?)`),
			itemID, jobID, c.ID, c.Title, c.Category, now)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	rememberEvalJobAuth(jobID, evalJobAuth{
		userID:   userID,
		source:   "eval",
		mcpToken: mcpToken,
	})
	job, items := h.loadEvalJob(r.Context(), db, userID, jobID)
	if job != nil {
		job["items"] = items
	}
	go h.runEvalJob(jobID)
	writeJSON(w, map[string]any{"ok": true, "job": job, "items": items})
}

func (h *Handler) evalJobsList(w http.ResponseWriter, r *http.Request) {
	db := h.evalDBOrError(w)
	if db == nil {
		return
	}
	if err := h.ensureEvalAutoTables(r.Context(), db); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	userID := resolveUserID(r)
	limit := parseLimit(r, 50, 200)
	rows, err := db.QueryContext(r.Context(), h.evalSQL(`
		SELECT id, suite_id, title, status, category_ids_json, case_ids_json, total, passed, failed,
		       error_text, started_at, ended_at, created_at
		FROM agent_eval_jobs
		WHERE user_id = ? OR (? = '' AND user_id = '')
		ORDER BY created_at DESC
		LIMIT ?`), userID, userID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		row, err := scanEvalJobRow(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, row)
	}
	writeJSON(w, map[string]any{"jobs": out, "total": len(out)})
}

func (h *Handler) evalJobGet(w http.ResponseWriter, r *http.Request) {
	db := h.evalDBOrError(w)
	if db == nil {
		return
	}
	if err := h.ensureEvalAutoTables(r.Context(), db); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	job, items := h.loadEvalJob(r.Context(), db, resolveUserID(r), id)
	if job == nil {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	job["items"] = items
	writeJSON(w, map[string]any{"job": job, "items": items})
}

func (h *Handler) evalJobCancel(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "job id required")
		return
	}
	cancelEvalJob(id)
	db := h.evalDBOrError(w)
	if db == nil {
		return
	}
	userID := resolveUserID(r)
	_, _ = db.ExecContext(r.Context(), h.evalSQL(`
		UPDATE agent_eval_jobs SET status = CASE WHEN status IN ('queued','running') THEN 'cancelled' ELSE status END
		WHERE id = ? AND (user_id = ? OR (? = '' AND user_id = ''))`), id, userID, userID)
	job, items := h.loadEvalJob(r.Context(), db, userID, id)
	if job == nil {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	job["items"] = items
	writeJSON(w, map[string]any{"ok": true, "job": job, "items": items})
}

func (h *Handler) evalJobItemDetail(w http.ResponseWriter, r *http.Request) {
	db := h.evalDBOrError(w)
	if db == nil {
		return
	}
	jobID := strings.TrimSpace(r.PathValue("id"))
	itemID := strings.TrimSpace(r.PathValue("item_id"))
	userID := resolveUserID(r)
	job, _ := h.loadEvalJob(r.Context(), db, userID, jobID)
	if job == nil {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	item := h.loadEvalJobItem(r.Context(), db, jobID, itemID)
	if item == nil {
		writeError(w, http.StatusNotFound, "item not found")
		return
	}
	sessionID, _ := item["session_id"].(string)
	payload := map[string]any{"item": item, "session": nil, "loop": nil}
	if sessionID == "" {
		writeJSON(w, payload)
		return
	}
	store, err := h.App.SessionStore()
	if err != nil {
		writeJSON(w, payload)
		return
	}
	chat, err := store.Load(sessionID)
	if err != nil || chat == nil {
		writeJSON(w, payload)
		return
	}
	if !chatsession.EnforceAccess(chat, userID) && userID != "" {
		writeError(w, http.StatusForbidden, "session access denied")
		return
	}
	payload["session"] = evalSessionSnapshot(chat)
	payload["loop"] = evalLoopSnapshot(chat)
	writeJSON(w, payload)
}

func evalSessionSnapshot(chat *chatsession.ChatSession) map[string]any {
	msgs := make([]map[string]any, 0, len(chat.Messages))
	for _, m := range chat.Messages {
		text := strings.TrimSpace(m.Content)
		if text == "" {
			continue
		}
		msgs = append(msgs, map[string]any{"role": string(m.Role), "content": text})
	}
	return map[string]any{
		"id":         chat.ID,
		"title":      chat.Title,
		"status":     chat.Status,
		"messages":   msgs,
		"updated_at": chat.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func evalLoopSnapshot(chat *chatsession.ChatSession) map[string]any {
	toolErrors := []map[string]any{}
	for _, rec := range chat.StepRecords {
		status := strings.ToLower(strings.TrimSpace(rec.ToolStatus))
		if rec.Kind == "tool" && (status == "error" || status == "failed" || strings.Contains(strings.ToLower(rec.Summary), "error")) {
			toolErrors = append(toolErrors, map[string]any{
				"step": rec.Step, "tool": rec.ToolName, "status": rec.ToolStatus, "summary": rec.Summary,
			})
		}
	}
	plan, _ := chatsession.LastTurnPlanFromSession(chat)
	return map[string]any{
		"step_records":   chat.StepRecords,
		"last_turn_plan": plan,
		"tools_called":   chatsession.LastTurnToolsCalledFromSession(chat),
		"tool_errors":    toolErrors,
	}
}

func (h *Handler) loadEvalJob(ctx context.Context, db *sql.DB, userID, id string) (map[string]any, []map[string]any) {
	if db == nil || strings.TrimSpace(id) == "" {
		return nil, nil
	}
	row := db.QueryRowContext(ctx, h.evalSQL(`
		SELECT id, suite_id, title, status, category_ids_json, case_ids_json, total, passed, failed,
		       error_text, started_at, ended_at, created_at
		FROM agent_eval_jobs
		WHERE id = ? AND (user_id = ? OR (? = '' AND user_id = ''))`), id, userID, userID)
	job, err := scanEvalJobRow(row)
	if err != nil {
		return nil, nil
	}
	rows, err := db.QueryContext(ctx, h.evalSQL(`
		SELECT id, case_id, title, category, status, session_id, run_id, detail, summary_json, checks_json,
		       loop_error, duration_ms, started_at, ended_at
		FROM agent_eval_job_items WHERE job_id = ? ORDER BY created_at ASC, case_id ASC`), id)
	if err != nil {
		return job, []map[string]any{}
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		item, err := scanEvalJobItemRow(rows)
		if err != nil {
			return job, nil
		}
		items = append(items, item)
	}
	return job, items
}

func (h *Handler) loadEvalJobItem(ctx context.Context, db *sql.DB, jobID, itemID string) map[string]any {
	row := db.QueryRowContext(ctx, h.evalSQL(`
		SELECT id, case_id, title, category, status, session_id, run_id, detail, summary_json, checks_json,
		       loop_error, duration_ms, started_at, ended_at
		FROM agent_eval_job_items WHERE job_id = ? AND id = ?`), jobID, itemID)
	item, err := scanEvalJobItemRow(row)
	if err != nil {
		return nil
	}
	return item
}

func scanEvalJobRow(s evalCaseScanner) (map[string]any, error) {
	var id, suiteID, title, status, catsJSON, casesJSON, errText, started, ended, created string
	var total, passed, failed int
	if err := s.Scan(&id, &suiteID, &title, &status, &catsJSON, &casesJSON, &total, &passed, &failed, &errText, &started, &ended, &created); err != nil {
		return nil, err
	}
	return map[string]any{
		"id":           id,
		"suite_id":     suiteID,
		"title":        title,
		"status":       status,
		"category_ids": decodeStringSlice(catsJSON),
		"case_ids":     decodeStringSlice(casesJSON),
		"total":        total,
		"passed":       passed,
		"failed":       failed,
		"error":        errText,
		"started_at":   started,
		"ended_at":     ended,
		"created_at":   created,
	}, nil
}

func scanEvalJobItemRow(s evalCaseScanner) (map[string]any, error) {
	var id, caseID, title, category, status, sessionID, runID, detail, summaryJSON, checksJSON, loopErr, started, ended string
	var duration sql.NullInt64
	if err := s.Scan(&id, &caseID, &title, &category, &status, &sessionID, &runID, &detail, &summaryJSON, &checksJSON, &loopErr, &duration, &started, &ended); err != nil {
		return nil, err
	}
	summary := map[string]any{}
	_ = json.Unmarshal([]byte(summaryJSON), &summary)
	checks := []any{}
	_ = json.Unmarshal([]byte(checksJSON), &checks)
	out := map[string]any{
		"id":         id,
		"case_id":    caseID,
		"title":      title,
		"category":   category,
		"status":     status,
		"session_id": sessionID,
		"run_id":     runID,
		"detail":     detail,
		"summary":    summary,
		"checks":     checks,
		"loop_error": loopErr,
		"started_at": started,
		"ended_at":   ended,
	}
	if duration.Valid {
		out["duration_ms"] = duration.Int64
	}
	return out, nil
}

func (h *Handler) ensureEvalAutoTables(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return nil
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS agent_eval_suites (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL DEFAULT '', title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '', category_ids_json TEXT NOT NULL DEFAULT '[]',
			case_ids_json TEXT NOT NULL DEFAULT '[]', created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS agent_eval_jobs (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL DEFAULT '', suite_id TEXT NOT NULL DEFAULT '',
			title TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'queued',
			category_ids_json TEXT NOT NULL DEFAULT '[]', case_ids_json TEXT NOT NULL DEFAULT '[]',
			total INTEGER NOT NULL DEFAULT 0, passed INTEGER NOT NULL DEFAULT 0, failed INTEGER NOT NULL DEFAULT 0,
			error_text TEXT NOT NULL DEFAULT '', started_at TEXT NOT NULL DEFAULT '', ended_at TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS agent_eval_job_items (
			id TEXT PRIMARY KEY, job_id TEXT NOT NULL, case_id TEXT NOT NULL, title TEXT NOT NULL DEFAULT '',
			category TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'pending', session_id TEXT NOT NULL DEFAULT '',
			run_id TEXT NOT NULL DEFAULT '', detail TEXT NOT NULL DEFAULT '', summary_json TEXT NOT NULL DEFAULT '{}',
			checks_json TEXT NOT NULL DEFAULT '[]', loop_error TEXT NOT NULL DEFAULT '', duration_ms INTEGER,
			started_at TEXT NOT NULL DEFAULT '', ended_at TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL)`,
	}
	if h.usesPostgresEval() {
		stmts = []string{
			`CREATE TABLE IF NOT EXISTS agent_eval_suites (
				id TEXT PRIMARY KEY, user_id TEXT NOT NULL DEFAULT '', title TEXT NOT NULL,
				description TEXT NOT NULL DEFAULT '', category_ids_json TEXT NOT NULL DEFAULT '[]',
				case_ids_json TEXT NOT NULL DEFAULT '[]', created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`,
			`CREATE TABLE IF NOT EXISTS agent_eval_jobs (
				id TEXT PRIMARY KEY, user_id TEXT NOT NULL DEFAULT '', suite_id TEXT NOT NULL DEFAULT '',
				title TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'queued',
				category_ids_json TEXT NOT NULL DEFAULT '[]', case_ids_json TEXT NOT NULL DEFAULT '[]',
				total INT NOT NULL DEFAULT 0, passed INT NOT NULL DEFAULT 0, failed INT NOT NULL DEFAULT 0,
				error_text TEXT NOT NULL DEFAULT '', started_at TEXT NOT NULL DEFAULT '', ended_at TEXT NOT NULL DEFAULT '',
				created_at TEXT NOT NULL)`,
			`CREATE TABLE IF NOT EXISTS agent_eval_job_items (
				id TEXT PRIMARY KEY, job_id TEXT NOT NULL, case_id TEXT NOT NULL, title TEXT NOT NULL DEFAULT '',
				category TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'pending', session_id TEXT NOT NULL DEFAULT '',
				run_id TEXT NOT NULL DEFAULT '', detail TEXT NOT NULL DEFAULT '', summary_json TEXT NOT NULL DEFAULT '{}',
				checks_json TEXT NOT NULL DEFAULT '[]', loop_error TEXT NOT NULL DEFAULT '', duration_ms INT,
				started_at TEXT NOT NULL DEFAULT '', ended_at TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL)`,
		}
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func decodeStringSlice(raw string) []string {
	out := []string{}
	if strings.TrimSpace(raw) == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	if out == nil {
		return []string{}
	}
	return out
}

func asStringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return uniqueNonEmpty(t)
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return uniqueNonEmpty(out)
	default:
		return []string{}
	}
}

func uniqueNonEmpty(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func containsFold(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func shortEvalID() string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), newEvalRunID("x")[len("run-x-"):])
}
