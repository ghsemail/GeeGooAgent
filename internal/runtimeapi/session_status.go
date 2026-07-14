package runtimeapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

const (
	defaultStatusPollMS = 500
	minStatusPollMS     = 200
	maxStatusPollMS     = 5000
	statusHeartbeatSec  = 30
)

// SessionStatusPayload is the debug snapshot for a chat session.
type SessionStatusPayload struct {
	SessionID    string                      `json:"session_id"`
	ResolvedFrom string                      `json:"resolved_from,omitempty"`
	Title        string                      `json:"title"`
	Status       string                      `json:"status"`
	Busy         bool                        `json:"busy"`
	LiveStatus   string                      `json:"live_status,omitempty"`
	MessageCount int                         `json:"message_count"`
	StepCount    int                         `json:"step_count"`
	UpdatedAt    time.Time                   `json:"updated_at"`
	Messages     []SessionMessageSummary     `json:"messages"`
	StepRecords  []chatsession.ChatStepRecord `json:"step_records"`
	Live         *chatsession.LiveSessionState `json:"live,omitempty"`
}

// SessionMessageSummary is a compact message row for remote debugging.
type SessionMessageSummary struct {
	Role             string `json:"role"`
	Content          string `json:"content,omitempty"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
	ToolCallCount    int    `json:"tool_call_count,omitempty"`
}

// Register mounts session debug routes on mux.
func (h *Handler) registerSessionRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/sessions/status", h.sessionStatus)
	mux.HandleFunc("GET /v1/sessions/status/stream", h.sessionStatusStream)
}

func (h *Handler) sessionStatus(w http.ResponseWriter, r *http.Request) {
	store, err := h.App.SessionStore()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	payload, code, msg := h.resolveSessionStatus(r, store)
	if code != http.StatusOK {
		writeError(w, code, msg)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *Handler) sessionStatusStream(w http.ResponseWriter, r *http.Request) {
	store, err := h.App.SessionStore()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	payload, code, msg := h.resolveSessionStatus(r, store)
	if code != http.StatusOK {
		writeError(w, code, msg)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	writeSessionSSE(w, flusher, "connected", map[string]any{
		"session_id": payload.SessionID,
		"resolved_from": payload.ResolvedFrom,
	})
	writeSessionSSE(w, flusher, "snapshot", payload)

	interval := pollIntervalMS(r)
	ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
	heartbeat := time.NewTicker(statusHeartbeatSec * time.Second)
	defer ticker.Stop()
	defer heartbeat.Stop()

	lastHash := hashSessionStatus(payload)
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			writeSessionSSE(w, flusher, "heartbeat", map[string]any{"ts": time.Now().UTC().Format(time.RFC3339Nano)})
		case <-ticker.C:
			updated, err := h.loadSessionStatus(store, payload.SessionID, payload.ResolvedFrom)
			if err != nil {
				writeSessionSSE(w, flusher, "error", map[string]string{"message": err.Error()})
				continue
			}
			if updated == nil {
				writeSessionSSE(w, flusher, "error", map[string]string{"message": "session not found"})
				return
			}
			hash := hashSessionStatus(updated)
			if hash == lastHash {
				continue
			}
			lastHash = hash
			writeSessionSSE(w, flusher, "update", updated)
		}
	}
}

func (h *Handler) resolveSessionStatus(r *http.Request, store chatsession.SessionStore) (*SessionStatusPayload, int, string) {
	sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
	resolvedFrom := "query"
	if sessionID == "" {
		id, err := latestSessionID(store)
		if err != nil {
			return nil, http.StatusInternalServerError, err.Error()
		}
		if id == "" {
			return nil, http.StatusNotFound, "no sessions found"
		}
		sessionID = id
		resolvedFrom = "latest"
	}
	payload, err := h.loadSessionStatus(store, sessionID, resolvedFrom)
	if err != nil {
		return nil, http.StatusInternalServerError, err.Error()
	}
	if payload == nil {
		return nil, http.StatusNotFound, "session not found: "+sessionID
	}
	return payload, http.StatusOK, ""
}

func (h *Handler) loadSessionStatus(store chatsession.SessionStore, sessionID, resolvedFrom string) (*SessionStatusPayload, error) {
	session, err := store.Load(sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}
	var live *chatsession.LiveSessionState
	if h.App != nil && h.App.State != nil {
		live, _ = chatsession.LoadLiveState(h.App.State, sessionID)
	}
	return buildSessionStatus(session, live, resolvedFrom), nil
}

func latestSessionID(store chatsession.SessionStore) (string, error) {
	entries, err := store.ListIndexedSessions()
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "", nil
	}
	return entries[0].ID, nil
}

func buildSessionStatus(session *chatsession.ChatSession, live *chatsession.LiveSessionState, resolvedFrom string) *SessionStatusPayload {
	payload := &SessionStatusPayload{
		SessionID:    session.ID,
		ResolvedFrom: resolvedFrom,
		Title:        session.Title,
		Status:       session.Status,
		MessageCount: len(session.Messages),
		StepCount:    len(session.StepRecords),
		UpdatedAt:    session.UpdatedAt,
		StepRecords:  append([]chatsession.ChatStepRecord(nil), session.StepRecords...),
		Live:         live,
	}
	if live != nil {
		payload.Busy = live.Busy
		payload.LiveStatus = live.Status
	}
	const maxMessages = 12
	start := 0
	if len(session.Messages) > maxMessages {
		start = len(session.Messages) - maxMessages
	}
	for _, msg := range session.Messages[start:] {
		payload.Messages = append(payload.Messages, summarizeMessage(msg))
	}
	return payload
}

func summarizeMessage(msg llm.Message) SessionMessageSummary {
	out := SessionMessageSummary{
		Role:             string(msg.Role),
		Content:          truncateRunes(msg.Content, 400),
		ReasoningContent: truncateRunes(msg.ReasoningContent, 240),
		ToolCallCount:    len(msg.ToolCalls),
	}
	return out
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

func pollIntervalMS(r *http.Request) int {
	raw := strings.TrimSpace(r.URL.Query().Get("interval_ms"))
	if raw == "" {
		return defaultStatusPollMS
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < minStatusPollMS {
		return defaultStatusPollMS
	}
	if v > maxStatusPollMS {
		return maxStatusPollMS
	}
	return v
}

func hashSessionStatus(payload *SessionStatusPayload) string {
	if payload == nil {
		return ""
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func writeSessionSSE(w http.ResponseWriter, flusher http.Flusher, event string, data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	if event != "" {
		_, _ = fmt.Fprintf(w, "event: %s\n", event)
	}
	_, _ = fmt.Fprintf(w, "data: %s\n\n", raw)
	flusher.Flush()
}
