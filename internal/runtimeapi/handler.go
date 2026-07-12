package runtimeapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
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
	h.registerSessionRoutes(mux)
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
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type streamChunk struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`
	Created int64               `json:"created"`
	Model   string              `json:"model"`
	Choices []streamChunkChoice `json:"choices"`
}

type streamChunkChoice struct {
	Index        int         `json:"index"`
	Delta        chatMessage `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
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
	if len(req.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "messages required")
		return
	}

	mcpToken := resolveMCPToken(r, req, h.App.Config.MCPToken())
	sessionID := "api-" + time.Now().Format("150405")
	ctx := h.App.ToolContext(sessionID)
	ctx.MCPToken = mcpToken
	ctx.Interactive = true
	if approveWrites(r) {
		ctx.Approved = true
	}

	upstream := make([]runtime.UpstreamMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		upstream = append(upstream, runtime.UpstreamMessage{Role: m.Role, Content: m.Content})
	}
	session, lastUser := runtime.NewUpstreamSession(upstream)
	if lastUser == "" {
		writeError(w, http.StatusBadRequest, "last message must be user")
		return
	}

	model := req.Model
	if model == "" {
		model = defaultModel
	}
	id := "chatcmpl-" + sessionID
	created := time.Now().Unix()
	schemas := h.App.Registry.Schemas(h.App.ChatToolNames())

	if req.Stream {
		h.streamChat(w, r, session, lastUser, ctx, schemas, id, model, created)
		return
	}

	result := h.App.Agent.Run(r.Context(), session, lastUser, ctx, schemas)
	finish := "stop"
	if result.Failed {
		finish = "error"
	}
	resp := chatResponse{
		ID: id, Object: "chat.completion", Created: created, Model: model,
		Choices: []chatChoice{{
			Index: 0, Message: chatMessage{Role: "assistant", Content: result.AssistantText}, FinishReason: finish,
		}},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) streamChat(
	w http.ResponseWriter,
	r *http.Request,
	session *runtime.Session,
	lastUser string,
	toolCtx tools.Context,
	schemas []llm.ToolSchema,
	id, model string,
	created int64,
) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	var (
		mu       sync.Mutex
		streamed atomic.Bool
	)
	writeChunk := func(delta chatMessage, finish *string) {
		mu.Lock()
		defer mu.Unlock()
		writeSSE(w, flusher, streamChunk{
			ID: id, Object: "chat.completion.chunk", Created: created, Model: model,
			Choices: []streamChunkChoice{{Index: 0, Delta: delta, FinishReason: finish}},
		})
	}

	writeChunk(chatMessage{Role: "assistant"}, nil)

	runCtx := llm.WithStreamHandler(r.Context(), func(delta llm.StreamDelta) {
		if delta.Content == "" {
			return
		}
		streamed.Store(true)
		writeChunk(chatMessage{Content: delta.Content}, nil)
	})

	result := h.App.Agent.Run(runCtx, session, lastUser, toolCtx, schemas)
	if !streamed.Load() && result.AssistantText != "" {
		for _, piece := range chunkText(result.AssistantText, 24) {
			writeChunk(chatMessage{Content: piece}, nil)
		}
	}
	finish := "stop"
	if result.Failed {
		finish = "error"
	}
	writeChunk(chatMessage{}, &finish)
	mu.Lock()
	_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
	mu.Unlock()
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, chunk streamChunk) {
	raw, err := json.Marshal(chunk)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "data: %s\n\n", raw)
	flusher.Flush()
}

func chunkText(s string, maxRunes int) []string {
	if s == "" {
		return nil
	}
	if maxRunes <= 0 {
		maxRunes = 24
	}
	var out []string
	var b strings.Builder
	n := 0
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		s = s[size:]
		b.WriteRune(r)
		n++
		if n >= maxRunes {
			out = append(out, b.String())
			b.Reset()
			n = 0
		}
	}
	if b.Len() > 0 {
		out = append(out, b.String())
	}
	return out
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

func approveWrites(r *http.Request) bool {
	v := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Approve-Writes")))
	return v == "1" || v == "true" || v == "yes"
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"message": msg},
	})
}
