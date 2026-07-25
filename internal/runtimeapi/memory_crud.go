package runtimeapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/memory/episodic"
	"github.com/ghsemail/GeeGooAgent/internal/memory/semantic"
)

func (h *Handler) registerMemoryCRUDRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/memory/episodes", h.memoryEpisodesList)
	mux.HandleFunc("POST /v1/memory/episodes", h.memoryEpisodeCreate)
	mux.HandleFunc("PUT /v1/memory/episodes/{id}", h.memoryEpisodeUpdate)
	mux.HandleFunc("DELETE /v1/memory/episodes/{id}", h.memoryEpisodeDelete)
	mux.HandleFunc("POST /v1/memory/chunks", h.memoryChunkCreate)
	mux.HandleFunc("PUT /v1/memory/chunks/{id}", h.memoryChunkUpdate)
	mux.HandleFunc("DELETE /v1/memory/chunks/{id}", h.memoryChunkDelete)
}

type episodePayload struct {
	SessionID  string `json:"session_id"`
	Summary    string `json:"summary"`
	HappenedAt string `json:"happened_at"`
}

type chunkPayload struct {
	SessionID string `json:"session_id"`
	Source    string `json:"source"`
	Content   string `json:"content"`
	Subject   string `json:"subject"`
}

func (h *Handler) memoryEpisodesList(w http.ResponseWriter, r *http.Request) {
	if h.App == nil || h.App.Episodic == nil {
		writeError(w, http.StatusServiceUnavailable, "episodic memory not enabled")
		return
	}
	userID := resolveUserID(r)
	limit := parseLimit(r, 50, 200)
	eps, err := h.App.Episodic.List(r.Context(), userID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]any{"episodes": episodeRows(eps), "total": len(eps)})
}

func (h *Handler) memoryEpisodeCreate(w http.ResponseWriter, r *http.Request) {
	if h.App == nil || h.App.Episodic == nil {
		writeError(w, http.StatusServiceUnavailable, "episodic memory not enabled")
		return
	}
	var req episodePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	userID := resolveUserID(r)
	when := parseDate(req.HappenedAt)
	id, err := h.App.Episodic.Create(r.Context(), strings.TrimSpace(req.SessionID), userID, req.Summary, when)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ep, _ := h.App.Episodic.GetByID(r.Context(), id)
	writeJSON(w, map[string]any{"ok": true, "episode": episodeRow(ep)})
}

func (h *Handler) memoryEpisodeUpdate(w http.ResponseWriter, r *http.Request) {
	if h.App == nil || h.App.Episodic == nil {
		writeError(w, http.StatusServiceUnavailable, "episodic memory not enabled")
		return
	}
	id, err := pathInt64(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid episode id")
		return
	}
	var req episodePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	when := parseDate(req.HappenedAt)
	if err := h.App.Episodic.Update(r.Context(), id, req.Summary, when); err != nil {
		writeEpisodeErr(w, err)
		return
	}
	ep, _ := h.App.Episodic.GetByID(r.Context(), id)
	writeJSON(w, map[string]any{"ok": true, "episode": episodeRow(ep)})
}

func (h *Handler) memoryEpisodeDelete(w http.ResponseWriter, r *http.Request) {
	if h.App == nil || h.App.Episodic == nil {
		writeError(w, http.StatusServiceUnavailable, "episodic memory not enabled")
		return
	}
	id, err := pathInt64(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid episode id")
		return
	}
	if err := h.App.Episodic.Delete(r.Context(), id); err != nil {
		writeEpisodeErr(w, err)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "deleted": id})
}

func (h *Handler) memoryChunkCreate(w http.ResponseWriter, r *http.Request) {
	if h.App == nil || h.App.Semantic == nil {
		writeError(w, http.StatusServiceUnavailable, "semantic memory not enabled")
		return
	}
	var req chunkPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	content := strings.TrimSpace(req.Content)
	subject := strings.TrimSpace(req.Subject)
	if subject != "" && !strings.HasPrefix(content, "[") {
		content = "[" + subject + "] " + content
	}
	userID := resolveUserID(r)
	id, err := h.App.Semantic.Create(r.Context(), strings.TrimSpace(req.SessionID), userID, req.Source, content)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	chunk, _ := h.App.Semantic.GetByID(r.Context(), id)
	writeJSON(w, map[string]any{"ok": true, "chunk": chunkRow(chunk)})
}

func (h *Handler) memoryChunkUpdate(w http.ResponseWriter, r *http.Request) {
	if h.App == nil || h.App.Semantic == nil {
		writeError(w, http.StatusServiceUnavailable, "semantic memory not enabled")
		return
	}
	id, err := pathInt64(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid chunk id")
		return
	}
	var req chunkPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	content := strings.TrimSpace(req.Content)
	subject := strings.TrimSpace(req.Subject)
	if subject != "" && !strings.HasPrefix(content, "[") {
		content = "[" + subject + "] " + content
	}
	if err := h.App.Semantic.UpdateContent(r.Context(), id, content); err != nil {
		writeChunkErr(w, err)
		return
	}
	chunk, _ := h.App.Semantic.GetByID(r.Context(), id)
	writeJSON(w, map[string]any{"ok": true, "chunk": chunkRow(chunk)})
}

func (h *Handler) memoryChunkDelete(w http.ResponseWriter, r *http.Request) {
	if h.App == nil || h.App.Semantic == nil {
		writeError(w, http.StatusServiceUnavailable, "semantic memory not enabled")
		return
	}
	id, err := pathInt64(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid chunk id")
		return
	}
	if err := h.App.Semantic.Delete(r.Context(), id); err != nil {
		writeChunkErr(w, err)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "deleted": id})
}

func episodeRows(eps []episodic.Episode) []map[string]any {
	out := make([]map[string]any, 0, len(eps))
	for i := range eps {
		out = append(out, episodeRow(&eps[i]))
	}
	return out
}

func episodeRow(ep *episodic.Episode) map[string]any {
	if ep == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":          ep.ID,
		"session_id":  ep.SessionID,
		"user_id":     ep.UserID,
		"summary":     ep.Summary,
		"happened_at": ep.HappenedAt.Format(time.RFC3339),
		"created_at":  ep.CreatedAt.Format(time.RFC3339),
		"source":      "episodic",
	}
}

func chunkRow(c *semantic.Chunk) map[string]any {
	if c == nil {
		return map[string]any{}
	}
	subject, content := splitFactContent(c.Content)
	return map[string]any{
		"id":         c.ID,
		"session_id": c.SessionID,
		"user_id":    c.UserID,
		"source":     c.Source,
		"subject":    subject,
		"content":    content,
		"raw":        c.Content,
		"created_at": c.CreatedAt.Format(time.RFC3339),
	}
}

func splitFactContent(raw string) (subject, content string) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "[") {
		return "", raw
	}
	end := strings.Index(raw, "]")
	if end <= 1 {
		return "", raw
	}
	subject = strings.TrimSpace(raw[1:end])
	content = strings.TrimSpace(raw[end+1:])
	return subject, content
}

func pathInt64(r *http.Request, key string) (int64, error) {
	raw := strings.TrimSpace(r.PathValue(key))
	return strconv.ParseInt(raw, 10, 64)
}

func parseDate(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Now().UTC()
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02", "2006-01-02T15:04:05Z07:00"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC()
		}
	}
	return time.Now().UTC()
}

func writeEpisodeErr(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "episode not found")
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}

func writeChunkErr(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "chunk not found")
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}
