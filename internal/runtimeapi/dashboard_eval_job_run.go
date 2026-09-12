package runtimeapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/eval"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

type evalJobAuth struct {
	userID   string
	source   string
	mcpToken string
}

type evalTurnOutcome struct {
	sessionID string
	reply     string
	failed    bool
	errText   string
}

var evalJobCtl = struct {
	mu     sync.Mutex
	auth   map[string]evalJobAuth
	cancel map[string]context.CancelFunc
}{
	auth:   map[string]evalJobAuth{},
	cancel: map[string]context.CancelFunc{},
}

// evalJobRunMu serializes live eval jobs so they do not interleave Agent.Run turns.
var evalJobRunMu sync.Mutex

func rememberEvalJobAuth(jobID string, auth evalJobAuth) {
	evalJobCtl.mu.Lock()
	defer evalJobCtl.mu.Unlock()
	evalJobCtl.auth[jobID] = auth
}

func takeEvalJobAuth(jobID string) (evalJobAuth, bool) {
	evalJobCtl.mu.Lock()
	defer evalJobCtl.mu.Unlock()
	auth, ok := evalJobCtl.auth[jobID]
	return auth, ok
}

func cancelEvalJob(jobID string) {
	evalJobCtl.mu.Lock()
	defer evalJobCtl.mu.Unlock()
	if fn := evalJobCtl.cancel[jobID]; fn != nil {
		fn()
	}
}

func registerEvalJobCancel(jobID string, fn context.CancelFunc) {
	evalJobCtl.mu.Lock()
	defer evalJobCtl.mu.Unlock()
	evalJobCtl.cancel[jobID] = fn
}

func clearEvalJobCtl(jobID string) {
	evalJobCtl.mu.Lock()
	defer evalJobCtl.mu.Unlock()
	delete(evalJobCtl.auth, jobID)
	delete(evalJobCtl.cancel, jobID)
}

func (h *Handler) runEvalJob(jobID string) {
	defer clearEvalJobCtl(jobID)
	db := h.dashboardSQLDB()
	if db == nil {
		return
	}
	auth, ok := takeEvalJobAuth(jobID)
	if !ok {
		h.markEvalJob(db, jobID, "error", "missing job credentials", 0, 0)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	registerEvalJobCancel(jobID, cancel)

	evalJobRunMu.Lock()
	defer evalJobRunMu.Unlock()
	if ctx.Err() != nil {
		h.markEvalJob(db, jobID, "cancelled", "cancelled before start", 0, 0)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = db.ExecContext(ctx, h.evalSQL(`UPDATE agent_eval_jobs SET status = 'running', started_at = ? WHERE id = ?`), now, jobID)

	items := h.listEvalJobItemIDs(ctx, db, jobID)
	passed, failed := 0, 0
	for _, item := range items {
		if ctx.Err() != nil {
			h.patchEvalJobItem(db, item.id, map[string]any{"status": "cancelled"})
			failed++
			continue
		}
		result := h.runEvalJobItem(ctx, db, auth, jobID, item)
		if result == "pass" {
			passed++
		} else {
			failed++
		}
	}
	status := "pass"
	if ctx.Err() != nil {
		status = "cancelled"
	} else if failed > 0 {
		status = "fail"
	}
	h.markEvalJob(db, jobID, status, "", passed, failed)
}

type evalJobItemRef struct {
	id       string
	caseID   string
	title    string
	category string
}

func (h *Handler) listEvalJobItemIDs(ctx context.Context, db *sql.DB, jobID string) []evalJobItemRef {
	rows, err := db.QueryContext(ctx, h.evalSQL(`
		SELECT id, case_id, title, category FROM agent_eval_job_items WHERE job_id = ? ORDER BY created_at ASC, case_id ASC`), jobID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []evalJobItemRef{}
	for rows.Next() {
		var item evalJobItemRef
		if err := rows.Scan(&item.id, &item.caseID, &item.title, &item.category); err != nil {
			continue
		}
		out = append(out, item)
	}
	return out
}

func (h *Handler) runEvalJobItem(ctx context.Context, db *sql.DB, auth evalJobAuth, jobID string, item evalJobItemRef) string {
	start := time.Now()
	h.patchEvalJobItem(db, item.id, map[string]any{"status": "running", "started_at": start.UTC().Format(time.RFC3339)})

	opts, title, err := h.loadTurnPlanCaseOptions(nil, item.caseID)
	if err != nil {
		h.finishEvalJobItem(db, item.id, "error", "", "", err.Error(), "", start, nil, nil)
		return "error"
	}
	if title == "" {
		title = item.title
	}
	opts = opts.Normalize().SyncLegacyUtterances()
	clarifyDefaults := eval.ClarifyDefaultTexts(opts)
	regularTurns, clarifyTurns := eval.DialogueExecutionPlan(opts)
	splitClarify := eval.UsesSplitClarifyScript(opts)

	sessionID := ""
	var lastOut evalTurnOutcome
	for i, turn := range regularTurns {
		lastOut, err = h.runEvalChatTurn(ctx, auth, sessionID, turn.Text, clarifyDefaults, !splitClarify)
		if err != nil {
			h.finishEvalJobItem(db, item.id, "error", lastOut.sessionID, "", "dialogue["+strconv.Itoa(i)+"]: "+err.Error(), lastOut.errText, start, nil, nil)
			return "error"
		}
		sessionID = lastOut.sessionID
		if lastOut.failed {
			h.finishEvalJobItem(db, item.id, "fail", sessionID, "", lastOut.errText, lastOut.errText, start, nil, nil)
			return "fail"
		}
	}

	store, err := h.App.SessionStore()
	if err != nil {
		h.finishEvalJobItem(db, item.id, "error", sessionID, "", err.Error(), "", start, nil, nil)
		return "error"
	}
	chat, err := store.Load(sessionID)
	if err != nil || chat == nil {
		h.finishEvalJobItem(db, item.id, "error", sessionID, "", "session not found after chat", "", start, nil, nil)
		return "error"
	}
	if len(clarifyTurns) > 0 && eval.NeedsClarifyFollowup(chat, opts) {
		for i, turn := range clarifyTurns {
			lastOut, err = h.runEvalChatTurn(ctx, auth, sessionID, turn.Text, clarifyDefaults, false)
			if err != nil {
				h.finishEvalJobItem(db, item.id, "error", lastOut.sessionID, "", "clarify["+strconv.Itoa(i)+"]: "+err.Error(), lastOut.errText, start, nil, nil)
				return "error"
			}
			sessionID = lastOut.sessionID
			if lastOut.failed {
				h.finishEvalJobItem(db, item.id, "fail", sessionID, "", lastOut.errText, lastOut.errText, start, nil, nil)
				return "fail"
			}
			chat, err = store.Load(sessionID)
			if err != nil || chat == nil {
				h.finishEvalJobItem(db, item.id, "error", sessionID, "", "session not found after clarify follow-up", "", start, nil, nil)
				return "error"
			}
		}
	}
	if chat.Metadata == nil {
		chat.Metadata = map[string]any{}
	}
	chat.Metadata["eval_job_id"] = jobID
	chat.Metadata["eval_case_id"] = item.caseID
	_ = store.Save(chat)

	result := eval.VerifyTurnPlanLiveFull(ctx, chat, opts, h.evalJudge())
	runID := newEvalRunID(item.caseID)
	_ = h.persistEvalVerifyRun(ctx, db, auth.userID, runID, item.caseID, title, sessionID, result, time.Since(start).Milliseconds())
	status := "pass"
	if !result.Passed {
		status = "fail"
	}
	h.finishEvalJobItem(db, item.id, status, sessionID, runID, result.Detail, lastOut.errText, start, result.Summary, result.Checks)
	return status
}

func (h *Handler) runEvalChatTurn(ctx context.Context, auth evalJobAuth, sessionID, message string, clarifyDefaults []string, enableClarifyFn bool) (evalTurnOutcome, error) {
	out := evalTurnOutcome{sessionID: sessionID}
	if h == nil || h.App == nil || h.App.Agent == nil || h.App.Gateway == nil {
		return out, fmt.Errorf("agent runtime not ready")
	}
	store, err := h.App.SessionStore()
	if err != nil {
		return out, err
	}
	turnCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	h.chatMu.Lock()
	defer h.chatMu.Unlock()

	chat, _, code, msg := h.loadOrCreateChatSession(store, sessionID, auth.userID, auth.source)
	if code != 200 {
		return out, fmt.Errorf("%s", msg)
	}
	out.sessionID = chat.ID
	if chat.Metadata == nil {
		chat.Metadata = map[string]any{}
	}
	chat.Metadata["source"] = "eval"

	h.App.Agent.SetProgress(nil)
	h.App.Agent.SetApproval(func(string, map[string]any) bool { return true })
	h.App.Agent.SetPlanGate(false)
	defer func() {
		h.App.Agent.SetApproval(nil)
		if h.App.Config != nil {
			h.App.Agent.SetPlanGate(h.App.Config.EffectivePlanGate())
		}
	}()

	chat.SyncChatSystemPrompt()
	rtSession := agent.RuntimeSessionFromChat(chat)
	toolCtx := h.App.ToolContextWithContext(turnCtx, chat.ID)
	toolCtx.UserID = auth.userID
	toolCtx.MCPToken = auth.mcpToken
	toolCtx.Interactive = false
	toolCtx.Approved = true
	if enableClarifyFn {
		toolCtx.ClarifyFn = func(ctx context.Context, question string, choices []string) (string, bool) {
			rec := eval.RecommendClarifyChoice(ctx, question, choices, eval.ClarifyRecommendContext{
				ClarifyDefaults: clarifyDefaults,
			}, h.clarifyRecommender())
			if answer, ok := rec.AnswerChoice(choices); ok {
				return answer, true
			}
			return eval.PickClarifyAnswer(question, choices, clarifyDefaults)
		}
	}
	toolSchemas := h.App.Registry.Schemas(h.App.ChatToolNames())
	var result runtime.TurnResult
	h.withUserAgentGateway(auth.userID, auth.source, func() {
		result = h.App.Agent.Run(turnCtx, rtSession, message, toolCtx, toolSchemas)
	})
	newRecords := stepRecordsFromTurn(result.StepRecords)
	agent.SyncChatFromRuntime(chat, rtSession, newRecords)
	if err := store.Save(chat); err != nil {
		return out, err
	}
	out.reply = result.AssistantText
	out.failed = result.Failed
	out.errText = result.Error
	return out, nil
}

func (h *Handler) finishEvalJobItem(
	db *sql.DB, itemID, status, sessionID, runID, detail, loopErr string,
	start time.Time, summary map[string]any, checks []eval.EvalCheckResult,
) {
	if summary == nil {
		summary = map[string]any{}
	}
	if checks == nil {
		checks = []eval.EvalCheckResult{}
	}
	summaryJSON, _ := json.Marshal(summary)
	checksJSON, _ := json.Marshal(checks)
	h.patchEvalJobItem(db, itemID, map[string]any{
		"status":       status,
		"session_id":   sessionID,
		"run_id":       runID,
		"detail":       detail,
		"summary_json": string(summaryJSON),
		"checks_json":  string(checksJSON),
		"loop_error":   loopErr,
		"duration_ms":  time.Since(start).Milliseconds(),
		"ended_at":     time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handler) patchEvalJobItem(db *sql.DB, itemID string, fields map[string]any) {
	if db == nil || itemID == "" || len(fields) == 0 {
		return
	}
	cols := []string{}
	args := []any{}
	for _, key := range []string{"status", "session_id", "run_id", "detail", "summary_json", "checks_json", "loop_error", "duration_ms", "started_at", "ended_at"} {
		v, ok := fields[key]
		if !ok {
			continue
		}
		cols = append(cols, key+" = ?")
		args = append(args, v)
	}
	if len(cols) == 0 {
		return
	}
	args = append(args, itemID)
	_, _ = db.Exec(h.evalSQL(`UPDATE agent_eval_job_items SET `+strings.Join(cols, ", ")+` WHERE id = ?`), args...)
}

func (h *Handler) markEvalJob(db *sql.DB, jobID, status, errText string, passed, failed int) {
	if db == nil {
		return
	}
	ended := time.Now().UTC().Format(time.RFC3339)
	_, _ = db.Exec(h.evalSQL(`
		UPDATE agent_eval_jobs SET status = ?, error_text = ?, passed = ?, failed = ?, ended_at = ? WHERE id = ?`),
		status, errText, passed, failed, ended, jobID)
}
