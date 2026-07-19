package runtimeapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
)

type planRequest struct {
	SessionID string `json:"session_id"`
	Approve   bool   `json:"approve"`
	Reject    bool   `json:"reject"`
}

type planResponse struct {
	Status        string `json:"status"`
	SessionID     string `json:"session_id"`
	AssistantText string `json:"assistant_text,omitempty"`
	PlanPending   bool   `json:"plan_pending"`
	Failed        bool   `json:"failed"`
	Error         string `json:"error,omitempty"`
	StepCount     int    `json:"step_count"`
}

func (h *Handler) registerPlanRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/chat/plan", h.chatPlan)
}

func (h *Handler) chatPlan(w http.ResponseWriter, r *http.Request) {
	if h.App == nil || h.App.Gateway == nil {
		writeError(w, http.StatusServiceUnavailable, "LLM not configured")
		return
	}
	store, err := h.App.SessionStore()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	var req planRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
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
	if _, _, ok := chat.HeldPlanFromMetadata(); !ok {
		writeError(w, http.StatusNotFound, "no pending plan for session")
		return
	}

	h.chatMu.Lock()
	defer h.chatMu.Unlock()

	rtSession := agent.RuntimeSessionFromChat(chat)
	toolCtx := h.App.ToolContextWithContext(r.Context(), chat.ID)
	toolCtx.MCPToken = resolveMCPToken(r, chatRequest{}, h.App.Config.MCPToken())
	toolCtx.Interactive = true
	if approveWrites(r) {
		toolCtx.Approved = true
	}
	if h.App.Config != nil {
		h.App.Agent.SetPlanGate(h.App.Config.EffectivePlanGate())
	}
	if approveWrites(r) {
		h.App.Agent.SetApproval(func(string, map[string]any) bool { return true })
	} else {
		h.App.Agent.SetApproval(nil)
	}

	userText := "y"
	if req.Reject {
		userText = "n"
	} else if !req.Approve {
		writeError(w, http.StatusBadRequest, "approve or reject required")
		return
	}

	schemas := h.App.Registry.Schemas(h.App.ChatToolNames())
	result := h.App.Agent.Run(r.Context(), rtSession, userText, toolCtx, schemas)

	newRecords := stepRecordsFromTurn(result.StepRecords)
	agent.SyncChatFromRuntime(chat, rtSession, newRecords)
	if err := store.Save(chat); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(planResponse{
		Status:        "ok",
		SessionID:     chat.ID,
		AssistantText: result.AssistantText,
		PlanPending:   result.PlanPending,
		Failed:        result.Failed,
		Error:         result.Error,
		StepCount:     len(chat.StepRecords),
	})
}
