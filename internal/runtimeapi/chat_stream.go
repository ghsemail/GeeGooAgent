package runtimeapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

const (
	defaultEventsPollMS = 200
	minEventsPollMS     = 100
	maxEventsPollMS     = 2000
	eventsHeartbeatSec  = 30
)

type chatStreamRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
	MCPToken  string `json:"mcp_token"`
}

type chatTurnEndPayload struct {
	SessionID     string `json:"session_id"`
	AssistantText string `json:"assistant_text,omitempty"`
	Failed        bool   `json:"failed"`
	Error         string `json:"error,omitempty"`
	StepCount     int    `json:"step_count"`
	PlanPending   bool   `json:"plan_pending"`
}

func (h *Handler) registerChatStreamRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/chat/stream", h.chatStream)
	mux.HandleFunc("GET /v1/sessions/events/stream", h.sessionEventsStream)
}

// chatStream runs one persisted chat turn and streams ReAct progress over SSE.
func (h *Handler) chatStream(w http.ResponseWriter, r *http.Request) {
	if h.App == nil || h.App.Gateway == nil {
		writeError(w, http.StatusServiceUnavailable, "LLM not configured")
		return
	}
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

	var req chatStreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		writeError(w, http.StatusBadRequest, "message required")
		return
	}

	h.chatMu.Lock()
	defer h.chatMu.Unlock()

	chat, created, code, msg := h.loadOrCreateChatSession(store, strings.TrimSpace(req.SessionID))
	if code != http.StatusOK {
		writeError(w, code, msg)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	resolvedFrom := "query"
	if created {
		resolvedFrom = "created"
	}
	writeSessionSSE(w, flusher, "connected", map[string]any{
		"session_id":    chat.ID,
		"resolved_from": resolvedFrom,
	})

	live := chatsession.NewLivePublisher(h.App.State, chat.ID)
	emit := func(event string, data map[string]any) {
		if live != nil {
			live.Emit(event, data)
		}
		writeSessionSSE(w, flusher, event, data)
	}

	h.App.Agent.SetProgress(func(event string, data map[string]any) {
		emit(event, data)
	})
	if approveWrites(r) {
		h.App.Agent.SetApproval(func(string, map[string]any) bool { return true })
	} else {
		h.App.Agent.SetApproval(nil)
	}
	defer h.App.Agent.SetProgress(nil)

	chat.SyncChatSystemPrompt()
	rtSession := agent.RuntimeSessionFromChat(chat)
	mcpToken := resolveMCPToken(r, chatRequest{MCPToken: req.MCPToken}, h.App.Config.MCPToken())
	toolCtx := h.App.ToolContextWithContext(r.Context(), chat.ID)
	toolCtx.MCPToken = mcpToken
	toolCtx.Interactive = true
	toolCtx.Approved = approveWrites(r)
	clarifyNotify := func(p PendingClarify) {
		emit("clarify", map[string]any{
			"session_id": p.SessionID,
			"question":   p.Question,
			"choices":    p.Choices,
		})
	}
	toolCtx.ClarifyFn = h.clarifyFn(r.Context(), chat.ID, clarifyNotify)
	if h.App.Config != nil {
		h.App.Agent.SetPlanGate(h.App.Config.EffectivePlanGate())
	}

	schemas := h.App.Registry.Schemas(h.App.ChatToolNames())
	result := h.App.Agent.Run(r.Context(), rtSession, message, toolCtx, schemas)

	newRecords := stepRecordsFromTurn(result.StepRecords)
	agent.SyncChatFromRuntime(chat, rtSession, newRecords)
	_ = store.Save(chat)
	if h.App.Semantic != nil && strings.TrimSpace(chat.Summary) != "" {
		userID := ""
		if chat.Metadata != nil {
			if v, ok := chat.Metadata["user_id"].(string); ok {
				userID = v
			}
		}
		_ = h.App.Semantic.UpsertSummary(r.Context(), chat.ID, userID, chat.Summary)
	}
	if live != nil {
		live.EndTurn()
	}

	writeSessionSSE(w, flusher, "turn_end", chatTurnEndPayload{
		SessionID:     chat.ID,
		AssistantText: result.AssistantText,
		Failed:        result.Failed,
		Error:         result.Error,
		StepCount:     len(chat.StepRecords),
		PlanPending:   result.PlanPending,
	})
	writeSessionSSE(w, flusher, "done", map[string]string{"session_id": chat.ID})
}

// sessionEventsStream pushes incremental live progress events for a session.
func (h *Handler) sessionEventsStream(w http.ResponseWriter, r *http.Request) {
	if h.App == nil || h.App.State == nil {
		writeError(w, http.StatusServiceUnavailable, "state store not configured")
		return
	}
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

	sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
	resolvedFrom := "query"
	if sessionID == "" {
		id, err := latestSessionID(store)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if id == "" {
			writeError(w, http.StatusNotFound, "no sessions found")
			return
		}
		sessionID = id
		resolvedFrom = "latest"
	} else if chat, err := store.Load(sessionID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	} else if chat == nil {
		writeError(w, http.StatusNotFound, "session not found: "+sessionID)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	writeSessionSSE(w, flusher, "connected", map[string]any{
		"session_id":    sessionID,
		"resolved_from": resolvedFrom,
	})

	interval := eventsPollIntervalMS(r)
	ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
	heartbeat := time.NewTicker(eventsHeartbeatSec * time.Second)
	defer ticker.Stop()
	defer heartbeat.Stop()

	seen := 0
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			writeSessionSSE(w, flusher, "heartbeat", map[string]any{"ts": time.Now().UTC().Format(time.RFC3339Nano)})
		case <-ticker.C:
			live, err := chatsession.LoadLiveState(h.App.State, sessionID)
			if err != nil {
				writeSessionSSE(w, flusher, "error", map[string]string{"message": err.Error()})
				continue
			}
			if live == nil {
				continue
			}
			for i := seen; i < len(live.Events); i++ {
				ev := live.Events[i]
				writeSessionSSE(w, flusher, "progress", map[string]any{
					"index": i,
					"event": ev.Event,
					"data":  ev.Data,
					"at":    ev.At.Format(time.RFC3339Nano),
				})
			}
			seen = len(live.Events)
		}
	}
}

func (h *Handler) loadOrCreateChatSession(store chatsession.SessionStore, sessionID string) (*chatsession.ChatSession, bool, int, string) {
	if sessionID != "" {
		chat, err := store.Load(sessionID)
		if err != nil {
			return nil, false, http.StatusInternalServerError, err.Error()
		}
		if chat == nil {
			return nil, false, http.StatusNotFound, "session not found: " + sessionID
		}
		return chat, false, http.StatusOK, ""
	}
	chat, err := store.Create()
	if err != nil {
		return nil, false, http.StatusInternalServerError, err.Error()
	}
	return chat, true, http.StatusOK, ""
}

func (h *Handler) runtimeSessionFromChat(chat *chatsession.ChatSession) *runtime.Session {
	return agent.RuntimeSessionFromChat(chat)
}

func stepRecordsFromTurn(records []runtime.StepRecord) []chatsession.ChatStepRecord {
	out := make([]chatsession.ChatStepRecord, 0, len(records))
	for _, rec := range records {
		out = append(out, chatsession.ChatStepRecord{
			Step: rec.Step, Timestamp: rec.Timestamp, Kind: rec.Kind,
			ToolName: rec.ToolName, ToolStatus: rec.ToolStatus, Summary: rec.Summary,
		})
	}
	return out
}

func eventsPollIntervalMS(r *http.Request) int {
	raw := strings.TrimSpace(r.URL.Query().Get("interval_ms"))
	if raw == "" {
		return defaultEventsPollMS
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < minEventsPollMS {
		return defaultEventsPollMS
	}
	if v > maxEventsPollMS {
		return maxEventsPollMS
	}
	return v
}
