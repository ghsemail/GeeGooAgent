package runtimeapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/taskflow"
)

type flowResumeRequest struct {
	SessionID string `json:"session_id"`
	MCPToken  string `json:"mcp_token"`
	Mode      string `json:"mode"` // continue | retry_failed
}

func (h *Handler) registerFlowRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/chat/flow/resume", h.chatFlowResume)
}

func (h *Handler) chatFlowResume(w http.ResponseWriter, r *http.Request) {
	if h.App == nil || h.App.Gateway == nil {
		writeError(w, http.StatusServiceUnavailable, "LLM not configured")
		return
	}
	store, err := h.App.SessionStore()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	var req flowResumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if !requireMCPTokenForChat(w, r, req.MCPToken, h.configMCPToken()) {
		return
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session_id required")
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
	userID := resolveUserID(r)
	if !enforceSessionAccess(w, chat, userID) {
		return
	}
	if taskflow.RawFromChat(chat) == nil {
		writeError(w, http.StatusConflict, "no active flow")
		return
	}
	message := "继续"
	if strings.EqualFold(strings.TrimSpace(req.Mode), "retry_failed") {
		message = "重试失败"
	}
	h.chatMu.Lock()
	defer h.chatMu.Unlock()

	chat.SyncChatSystemPrompt()
	rtSession := agent.RuntimeSessionFromChat(chat)
	toolCtx := h.App.ToolContextWithContext(r.Context(), chat.ID)
	toolCtx.UserID = userID
	toolCtx.MCPToken = resolveChatMCPToken(r, req.MCPToken, h.configMCPToken())
	toolCtx.Interactive = true
	toolCtx.Approved = approveWrites(r)
	schemas := h.App.Registry.Schemas(h.App.ChatToolNames())
	var result runtime.TurnResult
	source := resolveClientSource(r)
	h.withUserAgentGateway(userID, source, func() {
		result = h.App.Agent.Run(r.Context(), rtSession, message, toolCtx, schemas)
	})
	newRecords := stepRecordsFromTurn(result.StepRecords)
	agent.SyncChatFromRuntime(chat, rtSession, newRecords)
	if err := store.Save(chat); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	writeJSON(w, map[string]any{
		"session_id":     chat.ID,
		"assistant_text": result.AssistantText,
		"failed":         result.Failed,
		"error":          result.Error,
		"active_flow":    taskflow.RawFromChat(chat),
	})
}
