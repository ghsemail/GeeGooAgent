package runtimeapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/memory/exportmarkdown"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

const (
	clarifyAutoPickWait = 15 * time.Second
	defaultEventsPollMS = 200
	minEventsPollMS     = 100
	maxEventsPollMS     = 2000
	eventsHeartbeatSec  = 30
)

type chatStreamRequest struct {
	Message          string   `json:"message"`
	SessionID        string   `json:"session_id"`
	MCPToken         string   `json:"mcp_token"`
	ContextProfiles  []string `json:"context_profiles,omitempty"`
	ActiveScopes     []string `json:"active_scopes,omitempty"`
}

type chatTurnEndPayload struct {
	SessionID     string `json:"session_id"`
	AssistantText string `json:"assistant_text,omitempty"` // normalized Markdown; Web clients should replace stream_delta accumulation with this
	ContentFormat string `json:"content_format,omitempty"` // "markdown"
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
	h.App.RefreshOpsEmbedding(false)
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
	if !requireMCPTokenForChat(w, r, req.MCPToken, h.configMCPToken()) {
		return
	}

	h.chatMu.Lock()
	defer h.chatMu.Unlock()

	chat, created, code, msg := h.loadOrCreateChatSession(store, strings.TrimSpace(req.SessionID), resolveUserID(r), resolveClientSource(r))
	if code != http.StatusOK {
		writeError(w, code, msg)
		return
	}
	if len(req.ContextProfiles) > 0 || len(req.ActiveScopes) > 0 {
		chatsession.SetActiveScopes(chat, chatsession.MergeContextProfiles(
			chatsession.ActiveScopesFromSession(chat),
			append(req.ContextProfiles, req.ActiveScopes...),
		))
		_ = store.Save(chat)
	}

	setSessionSSEHeaders(w)
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
		writeAgentProgressSSE(w, flusher, event, data)
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
	mcpToken := resolveChatMCPToken(r, req.MCPToken, h.configMCPToken())
	toolCtx := h.App.ToolContextWithContext(r.Context(), chat.ID)
	toolCtx.UserID = resolveUserID(r)
	toolCtx.MCPToken = mcpToken
	toolCtx.Interactive = true
	toolCtx.Approved = approveWrites(r)
	clarifyHooks := newChatStreamClarifyHooks(emit)
	userID := resolveUserID(r)
	source := resolveClientSource(r)
	progressFn := func(event string, data map[string]any) { emit(event, data) }
	toolCtx.ClarifyFn = func(waitCtx context.Context, question string, choices []string) (string, bool) {
		ctx := waitCtx
		if ctx == nil {
			ctx = r.Context()
		}
		// Waiting for the user must not freeze every other chat behind chatMu.
		h.App.Agent.SetProgress(nil)
		h.chatMu.Unlock()
		defer func() {
			h.chatMu.Lock()
			if gw := h.userGateway(userID, source); gw != nil {
				h.App.Agent.SetGateway(gw)
			}
			h.App.Agent.SetProgress(progressFn)
		}()
		waitCtx, cancel := context.WithTimeout(ctx, clarifyAutoPickWait)
		defer cancel()
		return h.waitClarifyOrAuto(waitCtx, chat.ID, question, choices, clarifyHooks)
	}
	if h.App.Config != nil {
		h.App.Agent.SetPlanGate(h.App.Config.EffectivePlanGate())
	}

	schemas := h.App.Registry.Schemas(h.App.ChatToolNames())
	// Respect client disconnect (stop button / tab close) so chatMu is not held forever.
	runCtx := r.Context()
	var result runtime.TurnResult
	h.withUserAgentGateway(userID, source, func() {
		result = h.App.Agent.Run(runCtx, rtSession, message, toolCtx, schemas)
	})

	newRecords := stepRecordsFromTurn(result.StepRecords)
	agent.SyncChatFromRuntime(chat, rtSession, newRecords)
	if err := store.Save(chat); err != nil {
		slog.Error("chat stream save session failed", "session_id", chat.ID, "error", err)
		emit("save_error", map[string]any{"session_id": chat.ID, "message": err.Error()})
	}
	_ = h.persistTurnMemory(runCtx, chat, userID)
	if h.App.Consolidator != nil {
		if res, err := h.App.Consolidator.MaybeConsolidate(runCtx, chat); err == nil && (res.Facts > 0 || res.Episode) {
			if err := store.Save(chat); err != nil {
				slog.Error("chat stream save after consolidation failed", "session_id", chat.ID, "error", err)
			}
			writeSessionSSE(w, flusher, "consolidation", map[string]any{
				"session_id": chat.ID,
				"facts":      res.Facts,
				"episode":    res.Episode,
				"kind":       "distill",
			})
		}
	}
	if h.App != nil && (h.App.Facts != nil || h.App.Episodic != nil) {
		_ = exportmarkdown.Export(r.Context(), config.Home(), userID, h.App.Facts, h.App.Episodic)
	}
	if live != nil {
		live.EndTurn()
	}

	writeSessionSSE(w, flusher, "turn_end", chatTurnEndPayload{
		SessionID:     chat.ID,
		AssistantText: result.AssistantText,
		ContentFormat: "markdown",
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

	userID := resolveUserID(r)
	sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
	resolvedFrom := "query"
	if sessionID == "" {
		id, err := latestSessionIDForUser(store, userID)
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
	} else if !enforceSessionAccess(w, chat, userID) {
		return
	}

	setSessionSSEHeaders(w)
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
				payload := runtime.ProgressPayload(ev.Event, ev.Data)
				payload["index"] = i
				if !ev.At.IsZero() {
					payload["at"] = ev.At.Format(time.RFC3339Nano)
				}
				writeSessionSSE(w, flusher, "progress", payload)
			}
			seen = len(live.Events)
		}
	}
}

func (h *Handler) loadOrCreateChatSession(store chatsession.SessionStore, sessionID, userID, source string) (*chatsession.ChatSession, bool, int, string) {
	applyChannelMeta := func(chat *chatsession.ChatSession) bool {
		source = strings.TrimSpace(source)
		if source == "" || chat == nil {
			return false
		}
		if chat.Metadata == nil {
			chat.Metadata = map[string]any{}
		}
		if v, ok := chat.Metadata["source"].(string); ok && strings.TrimSpace(v) != "" {
			return false
		}
		chat.Metadata["source"] = NormalizeSessionSource(source)
		return true
	}

	if sessionID != "" {
		chat, err := store.Load(sessionID)
		if err != nil {
			return nil, false, http.StatusInternalServerError, err.Error()
		}
		if chat == nil {
			return nil, false, http.StatusNotFound, "session not found: " + sessionID
		}
		priorOwner := chatsession.UserIDFromSession(chat)
		if userID != "" && !chatsession.EnforceAccess(chat, userID) {
			return nil, false, http.StatusForbidden, "session access denied"
		}
		changed := applyChannelMeta(chat)
		if userID != "" && priorOwner == "" {
			changed = true
		}
		if changed {
			if err := store.Save(chat); err != nil {
				return nil, false, http.StatusInternalServerError, err.Error()
			}
		}
		return chat, false, http.StatusOK, ""
	}
	chat, err := store.Create()
	if err != nil {
		return nil, false, http.StatusInternalServerError, err.Error()
	}
	if userID != "" {
		chatsession.SetUserID(chat, userID)
	}
	applyChannelMeta(chat)
	if err := store.Save(chat); err != nil {
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
			PromptTokens: rec.PromptTokens, CompletionTokens: rec.CompletionTokens,
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
