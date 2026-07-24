package runtimeapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/doctor"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/memory/semantic"
)

func (h *Handler) registerCockpitRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/metrics/overview", h.metricsOverview)
	mux.HandleFunc("GET /v1/sessions", h.listSessions)
	mux.HandleFunc("GET /v1/sessions/{id}/trace", h.sessionTrace)
	mux.HandleFunc("GET /v1/tools", h.listTools)
	mux.HandleFunc("GET /v1/doctor", h.doctorStatus)
	mux.HandleFunc("GET /v1/memory/status", h.memoryStatus)
	mux.HandleFunc("GET /v1/memory/chunks", h.memoryChunks)
}

type metricsOverview struct {
	GeneratedAt    time.Time `json:"generated_at"`
	SessionCount   int       `json:"session_count"`
	MessageCount   int       `json:"message_count"`
	StepCount      int       `json:"step_count"`
	ToolCount      int       `json:"tool_count"`
	ChatToolCount  int       `json:"chat_tool_count"`
	LLMConfigured  bool      `json:"llm_configured"`
	SessionBackend string    `json:"session_backend"`
}

type sessionListItem struct {
	ID           string    `json:"id"`
	Title        string    `json:"title,omitempty"`
	Status       string    `json:"status"`
	MessageCount int       `json:"message_count"`
	StepCount    int       `json:"step_count"`
	UpdatedAt    time.Time `json:"updated_at"`
	Source       string    `json:"source,omitempty"`
}

type sessionListResponse struct {
	Sessions []sessionListItem `json:"sessions"`
	Total    int               `json:"total"`
}

type sessionTraceResponse struct {
	SessionID   string                      `json:"session_id"`
	Title       string                      `json:"title,omitempty"`
	Status      string                      `json:"status"`
	StepRecords []chatsession.ChatStepRecord `json:"step_records"`
	UpdatedAt   time.Time                   `json:"updated_at"`
}

type toolListItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ChatEnabled bool   `json:"chat_enabled"`
}

type toolsResponse struct {
	Tools []toolListItem `json:"tools"`
	Total int            `json:"total"`
}

type doctorCheckJSON struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Warn   bool   `json:"warn,omitempty"`
	Detail string `json:"detail"`
}

type doctorResponse struct {
	OK     bool              `json:"ok"`
	Checks []doctorCheckJSON `json:"checks"`
}

type memoryStatusResponse struct {
	SessionBackend string `json:"session_backend"`
	PostgresDSN    bool   `json:"postgres_configured"`
	PostgresOK     bool   `json:"postgres_ok"`
	VectorBackend  string `json:"vector_backend"`
	VectorEnabled  bool   `json:"vector_enabled"`
	Note           string `json:"note,omitempty"`
}

func (h *Handler) metricsOverview(w http.ResponseWriter, r *http.Request) {
	payload := metricsOverview{
		GeneratedAt:   time.Now().UTC(),
		LLMConfigured: h.App != nil && h.App.Gateway != nil,
		SessionBackend: sessionBackendName(h),
	}
	if h.App != nil && h.App.Registry != nil {
		payload.ToolCount = len(h.App.Registry.ListNames())
		chatNames := h.App.ChatToolNames()
		payload.ChatToolCount = len(chatNames)
	}
	store, err := h.App.SessionStore()
	if err == nil {
		entries, listErr := store.ListIndexedSessions()
		if listErr == nil {
			payload.SessionCount = len(entries)
			for _, e := range entries {
				payload.MessageCount += e.MessageCount
				payload.StepCount += e.StepCount
			}
		}
	}
	writeJSON(w, payload)
}

func (h *Handler) listSessions(w http.ResponseWriter, r *http.Request) {
	store, err := h.sessionStoreOrError(w)
	if err != nil {
		return
	}
	entries, err := store.ListIndexedSessions()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	limit := parseLimit(r, 50, 200)
	items := make([]sessionListItem, 0, len(entries))
	for i, e := range entries {
		if limit > 0 && i >= limit {
			break
		}
		source := ""
		if e.Metadata != nil {
			if v, ok := e.Metadata["source"].(string); ok {
				source = v
			}
		}
		items = append(items, sessionListItem{
			ID:           e.ID,
			Title:        e.Title,
			Status:       e.Status,
			MessageCount: e.MessageCount,
			StepCount:    e.StepCount,
			UpdatedAt:    e.UpdatedAt,
			Source:       source,
		})
	}
	writeJSON(w, sessionListResponse{Sessions: items, Total: len(entries)})
}

func (h *Handler) sessionTrace(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, sessionTraceResponse{
		SessionID:   session.ID,
		Title:       session.Title,
		Status:      session.Status,
		StepRecords: append([]chatsession.ChatStepRecord(nil), session.StepRecords...),
		UpdatedAt:   session.UpdatedAt,
	})
}

func (h *Handler) listTools(w http.ResponseWriter, r *http.Request) {
	if h.App == nil || h.App.Registry == nil {
		writeError(w, http.StatusServiceUnavailable, "tool registry not configured")
		return
	}
	chatSet := map[string]struct{}{}
	for _, name := range h.App.ChatToolNames() {
		chatSet[name] = struct{}{}
	}
	names := h.App.Registry.ListNames()
	items := make([]toolListItem, 0, len(names))
	for _, name := range names {
		tool, ok := h.App.Registry.Get(name)
		if !ok {
			continue
		}
		_, chatEnabled := chatSet[name]
		items = append(items, toolListItem{
			Name:        tool.Name,
			Description: tool.Description,
			ChatEnabled: chatEnabled,
		})
	}
	writeJSON(w, toolsResponse{Tools: items, Total: len(items)})
}

func (h *Handler) doctorStatus(w http.ResponseWriter, r *http.Request) {
	if h.App == nil || h.App.Config == nil {
		writeError(w, http.StatusServiceUnavailable, "config not loaded")
		return
	}
	skip := strings.EqualFold(r.URL.Query().Get("skip_connectivity"), "1") ||
		strings.EqualFold(r.URL.Query().Get("skip_connectivity"), "true")
	rows, ok := doctor.CollectFromConfig(h.App.Config, doctor.Options{SkipConnectivity: skip})
	checks := make([]doctorCheckJSON, 0, len(rows))
	for _, row := range rows {
		checks = append(checks, doctorCheckJSON{
			Name: row.Name, OK: row.OK, Warn: row.Warn, Detail: row.Detail,
		})
	}
	writeJSON(w, doctorResponse{OK: ok, Checks: checks})
}

func (h *Handler) memoryStatus(w http.ResponseWriter, r *http.Request) {
	backend := sessionBackendName(h)
	dsn := infra.PostgresDSN()
	resp := memoryStatusResponse{
		SessionBackend: backend,
		PostgresDSN:    dsn != "",
		VectorBackend:  "none",
		VectorEnabled:  false,
	}
	if dsn != "" {
		if err := infra.PingPostgres(dsn); err != nil {
			resp.Note = "PostgreSQL configured but unreachable: " + err.Error()
		} else {
			resp.PostgresOK = true
			resp.Note = "PostgreSQL reachable."
		}
	} else {
		resp.Note = "Session SSOT uses " + backend + ". Set GEEGOO_PG_DSN for PostgreSQL."
	}
	if h.App != nil && h.App.PG != nil && h.App.PG.MemorySchemaEnabled() {
		resp.VectorBackend = "pgvector"
		resp.VectorEnabled = true
		if h.App.Semantic != nil {
			if n, err := h.App.Semantic.Count(r.Context()); err == nil {
				resp.Note += fmt.Sprintf(" %d memory chunks.", n)
			}
		}
	} else if vectorEnvEnabled() {
		resp.Note += " GEEGOO_VECTOR_ENABLE=1 but pgvector schema not ready."
	}
	writeJSON(w, resp)
}

func vectorEnvEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("GEEGOO_VECTOR_ENABLE")))
	return v == "1" || v == "true" || v == "yes"
}

func (h *Handler) memoryChunks(w http.ResponseWriter, r *http.Request) {
	if h.App == nil || h.App.Semantic == nil {
		writeError(w, http.StatusServiceUnavailable, "semantic memory not enabled (GEEGOO_PG_DSN + GEEGOO_VECTOR_ENABLE)")
		return
	}
	limit := parseLimit(r, 30, 200)
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	var (
		chunks []semantic.Chunk
		err    error
	)
	if query != "" {
		chunks, err = h.App.Semantic.SearchVector(r.Context(), query, limit)
		if err != nil || len(chunks) == 0 {
			chunks, err = h.App.Semantic.SearchText(r.Context(), query, limit)
		}
	} else {
		chunks, err = h.App.Semantic.List(r.Context(), limit)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]any{"chunks": chunks, "total": len(chunks)})
}

func (h *Handler) sessionStoreOrError(w http.ResponseWriter) (chatsession.SessionStore, error) {
	if h.App == nil {
		writeError(w, http.StatusServiceUnavailable, "app not configured")
		return nil, errResponseWritten
	}
	store, err := h.App.SessionStore()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return nil, err
	}
	return store, nil
}

var errResponseWritten = errors.New("response already written")

func sessionBackendName(h *Handler) string {
	if h.App != nil {
		return h.App.SessionBackendName()
	}
	return "file"
}

func parseLimit(r *http.Request, defaultLimit, max int) int {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return defaultLimit
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return defaultLimit
	}
	if v > max {
		return max
	}
	return v
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}
