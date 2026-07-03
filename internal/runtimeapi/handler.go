package runtimeapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

const defaultModel = "geegoo-agent"

// Handler serves OpenAI-compatible endpoints for GeeGooBot agent-api (:3110).
type Handler struct {
	App *app.App
}

// NewHandler creates runtime API handlers.
func NewHandler(application *app.App) *Handler {
	return &Handler{App: application}
}

// Register mounts routes on mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/chat/completions", h.chatCompletions)
	mux.HandleFunc("GET /v1/models", h.listModels)
}

type chatRequest struct {
	Model    string          `json:"model"`
	Messages []chatMessage   `json:"messages"`
	Stream   bool            `json:"stream"`
	MCPToken string          `json:"mcp_token"`
	UserID   string          `json:"user_id"`
	Raw      json.RawMessage `json:"-"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []chatChoice `json:"choices"`
}

type chatChoice struct {
	Index        int          `json:"index"`
	Message      chatMessage  `json:"message"`
	FinishReason string       `json:"finish_reason"`
}

type modelsResponse struct {
	Object string      `json:"object"`
	Data   []modelItem `json:"data"`
}

type modelItem struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
}

func (h *Handler) chatCompletions(w http.ResponseWriter, r *http.Request) {
	if h.App == nil || h.App.Gateway == nil {
		writeError(w, http.StatusServiceUnavailable, "LLM not configured")
		return
	}
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Stream {
		writeError(w, http.StatusNotImplemented, "stream=true not implemented in A4; use stream=false")
		return
	}
	if len(req.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "messages required")
		return
	}

	mcpToken := resolveMCPToken(r, req, h.App.Config.MCPToken())
	sessionID := "api-" + time.Now().Format("150405")
	ctx := h.App.ToolContext(sessionID)
	ctx.MCPToken = mcpToken

	upstream := make([]runtime.UpstreamMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		upstream = append(upstream, runtime.UpstreamMessage{Role: m.Role, Content: m.Content})
	}
	session, lastUser := runtime.NewUpstreamSession(upstream)
	if lastUser == "" {
		writeError(w, http.StatusBadRequest, "last message must be user")
		return
	}

	schemas := h.App.Registry.Schemas(tools.RegisteredChatToolNames(h.App.Registry))
	result := h.App.Loop.RunTurn(r.Context(), session, lastUser, ctx, schemas)
	model := req.Model
	if model == "" {
		model = defaultModel
	}
	status := http.StatusOK
	content := result.AssistantText
	finish := "stop"
	if result.Failed {
		finish = "error"
	}
	resp := chatResponse{
		ID:      "chatcmpl-" + sessionID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []chatChoice{{
			Index:        0,
			Message:      chatMessage{Role: "assistant", Content: content},
			FinishReason: finish,
		}},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) listModels(w http.ResponseWriter, r *http.Request) {
	resp := modelsResponse{
		Object: "list",
		Data: []modelItem{{
			ID: defaultModel, Object: "model", OwnedBy: "geegoo",
		}},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func resolveMCPToken(r *http.Request, req chatRequest, fallback string) string {
	if v := strings.TrimSpace(r.Header.Get("X-MCP-Token")); v != "" {
		return v
	}
	if v := strings.TrimSpace(req.MCPToken); v != "" {
		return v
	}
	return fallback
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"message": msg},
	})
}
