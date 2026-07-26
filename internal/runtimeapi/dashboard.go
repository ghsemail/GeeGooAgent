package runtimeapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/doctor"
	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	factmem "github.com/ghsemail/GeeGooAgent/internal/memory/facts"
	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
	"github.com/ghsemail/GeeGooAgent/internal/skills"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func (h *Handler) registerDashboardRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/dashboard/data", h.dashboardData)
	mux.HandleFunc("GET /v1/dashboard/events", h.dashboardEvents)
	mux.HandleFunc("POST /v1/dashboard/query", h.dashboardQuery)
	mux.HandleFunc("GET /v1/dashboard/sessions/{id}/messages", h.dashboardSessionMessages)
	mux.HandleFunc("POST /v1/dashboard/voice", h.dashboardVoice)
	h.registerCompareRoutes(mux)
	h.registerSettingsRoutes(mux)
}

type dashboardTraceEvent struct {
	Type     string `json:"type"`
	Decision string `json:"decision,omitempty"`
	Detail   string `json:"detail,omitempty"`
	Session  string `json:"session_id,omitempty"`
	At       string `json:"ts,omitempty"`
}

type dashboardEventBus struct {
	mu     sync.Mutex
	events []dashboardTraceEvent
}

var globalDashEvents dashboardEventBus

var selectOnlySQL = regexp.MustCompile(`(?is)^\s*(with\b.*?)?\s*select\b`)

func (h *Handler) dashboardData(w http.ResponseWriter, r *http.Request) {
	payload, err := h.buildDashboardData(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, payload)
}

func (h *Handler) buildDashboardData(r *http.Request) (map[string]any, error) {
	now := time.Now().UTC()
	provider := "geegoo"
	model := defaultModel
	if h.App != nil && h.App.Gateway != nil {
		if m := strings.TrimSpace(h.App.Gateway.Model()); m != "" {
			model = m
		}
	}

	stats := map[string]any{
		"turns": 0, "tool_calls": 0, "gate_skips": 0, "gate_retrieves": 0,
		"latency_avg": 0, "trace_files": 0, "tool_errors": 0,
	}

	sessionsOut := []map[string]any{}
	turns := []map[string]any{}
	chatLog := []map[string]any{}
	facts := []map[string]any{}
	episodes := []map[string]any{}
	currentSession := ""
	totalIn, totalOut := 0, 0

	store, _ := h.safeSessionStore()
	userID := resolveUserID(r)
	if store != nil {
		entries, err := listSessionsForUser(store, userID)
		if err == nil {
			for i, e := range entries {
				if i == 0 {
					currentSession = e.ID
				}
				lastMsg := ""
				if sess, err := store.Load(e.ID); err == nil && sess != nil {
					for j := len(sess.Messages) - 1; j >= 0; j-- {
						if sess.Messages[j].Role != llm.RoleSystem {
							lastMsg = truncateRunes(sess.Messages[j].Content, 120)
							break
						}
					}
					for _, msg := range sess.Messages {
						if msg.Role == llm.RoleSystem {
							continue
						}
						chatLog = append(chatLog, map[string]any{
							"session_id": e.ID,
							"role":       string(msg.Role),
							"content":    truncateRunes(msg.Content, 200),
						})
					}
					for _, rec := range sess.StepRecords {
						totalIn += rec.PromptTokens
						totalOut += rec.CompletionTokens
					}
					turns = append(turns, buildTurnsFromSession(sess)...)
				}
				sources := []string{"web"}
				if e.Metadata != nil {
					if src, ok := e.Metadata["source"].(string); ok && src != "" {
						sources = []string{src}
					}
				}
				sessionsOut = append(sessionsOut, map[string]any{
					"id": e.ID, "title": firstNonEmpty(e.Title, e.ID),
					"messages": e.MessageCount, "steps": e.StepCount, "status": e.Status,
					"sources": sources, "last": lastMsg, "last_at": e.UpdatedAt.Format(time.RFC3339),
				})
				if summary := strings.TrimSpace(e.Summary); summary != "" {
					episodes = append(episodes, map[string]any{
						"session_id": e.ID, "title": firstNonEmpty(e.Title, e.ID),
						"summary": summary, "updated_at": e.UpdatedAt.Format(time.RFC3339),
						"message_count": e.MessageCount,
					})
				}
			}
		}
	}

	sort.Slice(turns, func(i, j int) bool {
		ti, _ := turns[i]["ts"].(string)
		tj, _ := turns[j]["ts"].(string)
		return ti > tj
	})
	if len(turns) > 30 {
		turns = turns[:30]
	}

	toolCalls, toolErrors := 0, 0
	latencies := []int{}
	for _, t := range turns {
		if raw, ok := t["tools"].([]map[string]any); ok {
			toolCalls += len(raw)
			for _, tool := range raw {
				if tool["status"] == "error" {
					toolErrors++
				}
			}
		}
		if ms, ok := t["latency_ms"].(int); ok && ms > 0 {
			latencies = append(latencies, ms)
		}
	}
	stats["turns"] = len(turns)
	stats["tool_calls"] = toolCalls
	stats["tool_errors"] = toolErrors
	if len(latencies) > 0 {
		sum := 0
		for _, v := range latencies {
			sum += v
		}
		stats["latency_avg"] = sum / len(latencies)
	}

	skillsOut, proceduralMemory := buildProceduralSkillsPayload(h.App)

	if h.App != nil && h.App.Episodic != nil {
		if eps, err := h.App.Episodic.List(r.Context(), userID, 80); err == nil && len(eps) > 0 {
			episodes = make([]map[string]any, 0, len(eps))
			for _, ep := range eps {
				episodes = append(episodes, map[string]any{
					"id":          ep.ID,
					"session_id":  ep.SessionID,
					"title":       truncateRunes(ep.Summary, 60),
					"summary":     ep.Summary,
					"happened_at": ep.HappenedAt.Format(time.RFC3339),
					"updated_at":  ep.HappenedAt.Format(time.RFC3339),
					"source":      "episodic",
				})
			}
		}
	}

	toolsPayload := map[string]any{
		"catalog": []map[string]any{}, "mcp": map[string]any{"configured": false, "servers": []string{}, "live": false},
		"apple_on": false, "planned": []map[string]any{},
		"toolsets": []tools.ToolsetSummary{}, "taxonomies": []tools.TaxonomySummary{},
	}
	if h.App != nil && h.App.Registry != nil {
		catalogItems := tools.BuildCatalog(h.App.Registry, h.App.ChatToolNames())
		catalog := make([]map[string]any, 0, len(catalogItems))
		for _, item := range catalogItems {
			catalog = append(catalog, tools.CatalogItemToMap(item))
		}
		toolsPayload["catalog"] = catalog
		toolsPayload["toolsets"] = tools.BuildToolsetSummaries()
		toolsPayload["taxonomies"] = tools.BuildTaxonomySummaries()
		if h.App.MCP != nil {
			toolsPayload["mcp"] = map[string]any{"configured": true, "servers": []string{"mcp"}, "live": true}
		}
	}

	if h.App != nil && h.App.Facts != nil {
		userID := resolveUserID(r)
		if rows, err := h.App.Facts.List(r.Context(), userID, 80); err == nil {
			for _, f := range rows {
				facts = append(facts, map[string]any{
					"id": f.ID, "subject": f.Subject, "content": f.Content,
					"raw": factmem.Format(f.Subject, f.Content),
					"source": f.Source, "user_id": f.UserID,
					"created_at": f.CreatedAt.Format(time.RFC3339),
				})
			}
		}
	}

	doctorOK := true
	doctorChecks := []map[string]any{}
	if h.App != nil && h.App.Config != nil {
		rows, _ := doctor.CollectFromConfig(h.App.Config, doctor.Options{SkipConnectivity: true})
		for _, row := range rows {
			doctorChecks = append(doctorChecks, map[string]any{
				"name": row.Name, "ok": row.OK, "warn": row.Warn, "detail": row.Detail,
			})
			if !row.OK && !row.Warn {
				doctorOK = false
			}
		}
	}

	home := ""
	if h.App != nil {
		home = h.App.Workspace
	}

	calendar := buildCalendarFromTurns(turns)
	totalCost := estimateUsageCostUSD(totalIn, totalOut)

	return map[string]any{
		"generated_at": now.Format(time.RFC3339), "provider": provider, "model": model,
		"small_model": model, "home": home, "current_session": currentSession, "stats": stats,
		"sessions": sessionsOut, "turns": turns, "chat_log": chatLog, "facts": facts,
		"episodes": episodes, "skills": skillsOut, "procedural_memory": proceduralMemory,
		"calendar": calendar, "outbox": []map[string]any{},
		"soul": soulTextForDashboard(firstNonEmpty(home, config.Home()), userID),
		"consolidate_every": 4, "chat_pending": 0, "tools": toolsPayload,
		"db": h.buildDBMeta(), "doctor_ok": doctorOK, "doctor_checks": doctorChecks,
		"eval_report": nil, "eval_history": []map[string]any{},
		"trace_tail": h.buildTraceTail(store, userID), "trace_file": "",
		"usage": map[string]any{
			"total_cost": totalCost, "calls": len(turns), "total_in": totalIn, "total_out": totalOut,
			"by_day": []map[string]any{}, "by_provider": []map[string]any{},
		},
		"settings": h.buildDashboardSettings(provider, model, userID), "wake_scans": []map[string]any{},
		"data_fleet": h.buildDataFleetSummary(r),
	}, nil
}

func (h *Handler) buildDataFleetSummary(r *http.Request) map[string]any {
	ctx, cancel := context.WithTimeout(r.Context(), dataBFFNodeTimeout+4*time.Second)
	defer cancel()
	payload, err := h.collectDataOverview(ctx, false)
	if err != nil {
		return map[string]any{"ok": false}
	}
	out := map[string]any{"ok": payload["ok"] == true}
	if summary, ok := payload["summary"].(map[string]any); ok {
		for _, key := range []string{"nodes_total", "nodes_healthy", "sources_total", "sources_healthy"} {
			if v, exists := summary[key]; exists {
				out[key] = v
			}
		}
	}
	return out
}

func (h *Handler) buildDashboardSettings(provider, model, userID string) map[string]any {
	info, err := h.buildSettingsInfo(userID)
	if err != nil {
		return map[string]any{
			"provider": provider, "model": model, "small_model": model,
			"pinned": []map[string]any{{"provider": provider, "model": model}},
		}
	}
	return info
}

func (h *Handler) buildDBMeta() map[string]any {
	tables := []map[string]any{}
	allTables := []string{}
	path := ""
	var db *sql.DB
	if h.App != nil && h.App.PG != nil {
		path = "postgresql"
		db = h.App.PG.SQL()
	} else if h.App != nil && h.App.DB != nil {
		path = "sqlite"
		db = h.App.DB.SQL()
	}
	if db != nil {
		isPG := path == "postgresql"
		names, err := listTableNames(db, isPG)
		if err == nil {
			allTables = names
			for _, name := range names {
				tables = append(tables, map[string]any{
					"name": name, "count": countTableRows(db, name),
					"columns": listTableColumns(db, name, isPG),
					"sample":  sampleTableRows(db, name, isPG, 5),
				})
			}
		}
	}
	return map[string]any{"path": path, "size": 0, "tables": tables, "all_tables": allTables, "fts": []string{}}
}

func listTableNames(db *sql.DB, postgres bool) ([]string, error) {
	var rows *sql.Rows
	var err error
	if postgres {
		rows, err = db.Query(`SELECT table_name FROM information_schema.tables WHERE table_schema='public' ORDER BY table_name`)
	} else {
		rows, err = db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil {
			out = append(out, name)
		}
	}
	return out, rows.Err()
}

func countTableRows(db *sql.DB, table string) int {
	if !safeIdent(table) {
		return 0
	}
	var n int
	_ = db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s`, quoteIdent(table))).Scan(&n)
	return n
}

func listTableColumns(db *sql.DB, table string, postgres bool) []string {
	if !safeIdent(table) {
		return nil
	}
	var rows *sql.Rows
	var err error
	if postgres {
		rows, err = db.Query(`SELECT column_name FROM information_schema.columns WHERE table_schema='public' AND table_name=$1 ORDER BY ordinal_position`, table)
	} else {
		rows, err = db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, quoteIdent(table)))
	}
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		if postgres {
			var c string
			if rows.Scan(&c) == nil {
				out = append(out, c)
			}
		} else {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dflt sql.NullString
			if rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk) == nil {
				out = append(out, name)
			}
		}
	}
	return out
}

func sampleTableRows(db *sql.DB, table string, postgres bool, limit int) []map[string]any {
	if !safeIdent(table) || limit <= 0 {
		return nil
	}
	rows, err := db.Query(fmt.Sprintf(`SELECT * FROM %s LIMIT %d`, quoteIdent(table), limit))
	if err != nil {
		return nil
	}
	defer rows.Close()
	colNames, _ := rows.Columns()
	out := []map[string]any{}
	for rows.Next() {
		vals := make([]any, len(colNames))
		ptrs := make([]any, len(colNames))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if rows.Scan(ptrs...) != nil {
			continue
		}
		row := map[string]any{}
		for i, c := range colNames {
			row[c] = stringifyCell(vals[i])
		}
		out = append(out, row)
	}
	return out
}

func stringifyCell(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case []byte:
		return string(t)
	case time.Time:
		return t.Format(time.RFC3339)
	default:
		return fmt.Sprint(v)
	}
}

var identRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func safeIdent(s string) bool { return identRe.MatchString(s) }

func quoteIdent(s string) string {
	if !safeIdent(s) {
		return `""`
	}
	return `"` + s + `"`
}

func buildTurnsFromSession(sess *chatsession.ChatSession) []map[string]any {
	if sess == nil {
		return nil
	}
	type turnMeta struct {
		tools    []map[string]any
		start    time.Time
		end      time.Time
		hasReply bool
	}
	metas := []turnMeta{}
	var cur *turnMeta
	for _, step := range sess.StepRecords {
		switch step.Kind {
		case "plan":
			metas = append(metas, turnMeta{start: step.Timestamp, tools: []map[string]any{}})
			cur = &metas[len(metas)-1]
		case "tool":
			if cur == nil {
				continue
			}
			status := "ok"
			if step.ToolStatus == "error" || step.ToolStatus == "failed" {
				status = "error"
			}
			cur.tools = append(cur.tools, map[string]any{
				"tool": step.ToolName, "summary": step.Summary, "status": status,
			})
		case "reply":
			if cur == nil {
				continue
			}
			cur.end = step.Timestamp
			cur.hasReply = true
			cur = nil
		}
	}

	turns := []map[string]any{}
	metaIdx := 0
	for i := 0; i < len(sess.Messages); i++ {
		if sess.Messages[i].Role != llm.RoleUser {
			continue
		}
		user := sess.Messages[i].Content
		reply := ""
		for j := i + 1; j < len(sess.Messages); j++ {
			if sess.Messages[j].Role == llm.RoleAssistant {
				reply = sess.Messages[j].Content
				i = j
				break
			}
		}
		t := map[string]any{
			"user_message": truncateRunes(user, 200),
			"reply":        truncateRunes(reply, 500),
			"tools":        []map[string]any{},
			"latency_ms":   0,
		}
		if metaIdx < len(metas) {
			m := metas[metaIdx]
			t["tools"] = m.tools
			if m.hasReply && !m.end.IsZero() && !m.start.IsZero() {
				ms := int(m.end.Sub(m.start).Milliseconds())
				if ms > 0 {
					t["latency_ms"] = ms
				}
			}
			if !m.start.IsZero() {
				t["ts"] = m.start.Format(time.RFC3339)
			}
			metaIdx++
		}
		turns = append(turns, t)
	}
	return turns
}

func buildCalendarFromTurns(turns []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(turns))
	for _, t := range turns {
		ts, _ := t["ts"].(string)
		if ts == "" {
			continue
		}
		title, _ := t["user_message"].(string)
		if title == "" {
			title = "agent turn"
		}
		out = append(out, map[string]any{
			"title": truncateRunes(title, 60),
			"at":    ts,
			"kind":  "turn",
		})
	}
	if len(out) > 30 {
		out = out[:30]
	}
	return out
}

func estimateUsageCostUSD(promptTokens, completionTokens int) float64 {
	if promptTokens <= 0 && completionTokens <= 0 {
		return 0
	}
	return float64(promptTokens)/1000*0.001 + float64(completionTokens)/1000*0.002
}

func (h *Handler) buildTraceTail(store chatsession.SessionStore, userID string) []map[string]any {
	out := []map[string]any{}
	if store == nil {
		return out
	}
	entries, err := listSessionsForUser(store, userID)
	if err != nil || len(entries) == 0 {
		return out
	}
	sess, err := store.Load(entries[0].ID)
	if err != nil || sess == nil {
		return out
	}
	start := 0
	if len(sess.StepRecords) > 12 {
		start = len(sess.StepRecords) - 12
	}
	for _, step := range sess.StepRecords[start:] {
		out = append(out, map[string]any{
			"type": step.Kind, "detail": firstNonEmpty(step.Summary, step.ToolName),
			"ts": step.Timestamp.Format(time.RFC3339),
		})
	}
	return out
}

func (h *Handler) dashboardEvents(w http.ResponseWriter, r *http.Request) {
	cursor := parseEventCursor(r)
	newEvents := h.collectLiveTraceEvents(resolveUserID(r))
	globalDashEvents.mu.Lock()
	for _, ev := range newEvents {
		globalDashEvents.events = append(globalDashEvents.events, ev)
	}
	if len(globalDashEvents.events) > 500 {
		globalDashEvents.events = globalDashEvents.events[len(globalDashEvents.events)-500:]
	}
	total := len(globalDashEvents.events)
	out := []dashboardTraceEvent{}
	if cursor != nil {
		if *cursor >= 0 && *cursor < total {
			out = append(out, globalDashEvents.events[*cursor:]...)
		}
	}
	globalDashEvents.mu.Unlock()
	if cursor == nil {
		out = nil
	}
	typed := make([]map[string]any, 0, len(out))
	for _, ev := range out {
		m := map[string]any{"type": ev.Type, "ts": ev.At, "session_id": ev.Session}
		if ev.Decision != "" {
			m["decision"] = ev.Decision
		}
		if ev.Detail != "" {
			m["detail"] = ev.Detail
		}
		typed = append(typed, m)
	}
	writeJSON(w, map[string]any{"events": typed, "cursor": total})
}

func parseEventCursor(r *http.Request) *int {
	raw := strings.TrimSpace(r.URL.Query().Get("cursor"))
	if raw == "" {
		return nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return nil
	}
	return &v
}

func (h *Handler) collectLiveTraceEvents(userID string) []dashboardTraceEvent {
	out := []dashboardTraceEvent{}
	if h.App == nil || h.App.State == nil {
		return out
	}
	store, err := h.safeSessionStore()
	if err != nil || store == nil {
		return out
	}
	entries, err := listSessionsForUser(store, userID)
	if err != nil {
		return out
	}
	limit := 5
	if len(entries) < limit {
		limit = len(entries)
	}
	for i := 0; i < limit; i++ {
		id := entries[i].ID
		live, err := chatsession.LoadLiveState(h.App.State, id)
		if err != nil || live == nil || len(live.Events) == 0 {
			continue
		}
		for _, ev := range live.Events {
			mapped := mapLiveEventToTrace(ev, id)
			if mapped.Type != "" {
				out = append(out, mapped)
			}
		}
	}
	return out
}

func mapLiveEventToTrace(ev chatsession.LiveEvent, sessionID string) dashboardTraceEvent {
	ts := ev.At.UTC().Format(time.RFC3339Nano)
	switch ev.Event {
	case "turn_start":
		return dashboardTraceEvent{Type: "turn_start", Session: sessionID, At: ts}
	case "tool_start", "tool_done", "tool_intercepted":
		return dashboardTraceEvent{Type: "tool", Session: sessionID, At: ts, Detail: ev.Event}
	case "turn_end":
		return dashboardTraceEvent{Type: "turn_end", Session: sessionID, At: ts}
	case "llm_plan", "thinking_start", "stream_delta", "llm_tools", "reply_start":
		return dashboardTraceEvent{Type: "llm", Session: sessionID, At: ts}
	case "gate":
		decision := ""
		if ev.Data != nil {
			decision, _ = ev.Data["decision"].(string)
		}
		return dashboardTraceEvent{Type: "gate", Session: sessionID, At: ts, Decision: decision}
	default:
		return dashboardTraceEvent{}
	}
}

type dashboardQueryRequest struct {
	SQL string `json:"sql"`
}

func (h *Handler) dashboardQuery(w http.ResponseWriter, r *http.Request) {
	var req dashboardQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	sqlText := strings.TrimSpace(req.SQL)
	if sqlText == "" {
		writeError(w, http.StatusBadRequest, "sql required")
		return
	}
	if !selectOnlySQL.MatchString(sqlText) {
		writeError(w, http.StatusBadRequest, "only SELECT queries are allowed")
		return
	}
	if strings.Contains(strings.ToUpper(sqlText), ";") {
		parts := strings.Split(sqlText, ";")
		nonEmpty := 0
		for _, p := range parts {
			if strings.TrimSpace(p) != "" {
				nonEmpty++
			}
		}
		if nonEmpty > 1 {
			writeError(w, http.StatusBadRequest, "only one statement allowed")
			return
		}
		sqlText = strings.TrimSuffix(sqlText, ";")
	}

	db := h.dashboardSQLDB()
	if db == nil {
		writeError(w, http.StatusServiceUnavailable, "no database configured (GEEGOO_PG_DSN or SQLite)")
		return
	}
	rows, err := db.Query(sqlText)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resultRows := [][]any{}
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		row := make([]any, len(cols))
		for i, v := range vals {
			row[i] = stringifyCell(v)
		}
		resultRows = append(resultRows, row)
	}
	writeJSON(w, map[string]any{"columns": cols, "rows": resultRows})
}

func (h *Handler) dashboardSQLDB() *sql.DB {
	if h.App == nil {
		return nil
	}
	if h.App.PG != nil {
		return h.App.PG.SQL()
	}
	if h.App.DB != nil {
		return h.App.DB.SQL()
	}
	return nil
}

func (h *Handler) dashboardSessionMessages(w http.ResponseWriter, r *http.Request) {
	store, err := h.sessionStoreOrError(w)
	if err != nil {
		return
	}
	sessionID := strings.TrimSpace(r.PathValue("id"))
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session id required")
		return
	}
	session, err := store.Load(sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if session == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if !enforceSessionAccess(w, session, resolveUserID(r)) {
		return
	}
	messages := []map[string]any{}
	for _, msg := range session.Messages {
		if msg.Role == llm.RoleSystem {
			continue
		}
		messages = append(messages, map[string]any{
			"role":              string(msg.Role),
			"content":           msg.Content,
			"reasoning_content": msg.ReasoningContent,
			"tool_call_count":   len(msg.ToolCalls),
		})
	}
	writeJSON(w, map[string]any{
		"session_id": session.ID,
		"title":      session.Title,
		"status":     session.Status,
		"messages":   messages,
		"updated_at": session.UpdatedAt,
	})
}

func (h *Handler) dashboardVoice(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "read body failed")
		return
	}
	if len(body) < 44 {
		writeJSON(w, map[string]any{"error": "audio too short"})
		return
	}
	// Voice transcription requires a local Whisper endpoint — not bundled in GeeGooAgent yet.
	_ = body
	writeJSON(w, map[string]any{
		"error": "voice transcription not configured on server — set GEEGOO_WHISPER_URL or use text input",
	})
}

func (h *Handler) safeSessionStore() (chatsession.SessionStore, error) {
	if h.App == nil {
		return nil, fmt.Errorf("app not configured")
	}
	return h.App.SessionStore()
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func buildProceduralSkillsPayload(app *app.App) ([]map[string]any, map[string]any) {
	var loader *procedural.Loader
	if app != nil {
		loader = app.SkillLoader
	}
	cfg := procedural.BuildMemoryConfig(loader, procedural.DefaultMaxSkillsPerTurn)
	out := make([]map[string]any, 0)
	if loader != nil {
		summaries := loader.ListSummaries()
		sort.Slice(summaries, func(i, j int) bool {
			if summaries[i].Kind != summaries[j].Kind {
				return summaries[i].Kind < summaries[j].Kind
			}
			return summaries[i].Name < summaries[j].Name
		})
		for _, s := range summaries {
			out = append(out, s.Map())
		}
	} else {
		for _, sk := range skills.Default().List() {
			out = append(out, map[string]any{
				"name": sk.Name, "description": sk.Description,
				"body": sk.Description, "body_preview": sk.Description,
				"path": sk.ManifestPath, "rel": sk.ManifestPath,
				"kind": string(procedural.KindWorkflow), "kind_label": procedural.KindLabel(procedural.KindWorkflow),
				"inject_in_chat": false, "editable": false,
			})
		}
	}
	return out, cfg.Map()
}
