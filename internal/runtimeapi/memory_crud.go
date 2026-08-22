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
	"github.com/ghsemail/GeeGooAgent/internal/memory/facts"
)

func (h *Handler) registerMemoryCRUDRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/memory/episodes", h.memoryEpisodesList)
	mux.HandleFunc("POST /v1/memory/episodes", h.memoryEpisodeCreate)
	mux.HandleFunc("PUT /v1/memory/episodes/{id}", h.memoryEpisodeUpdate)
	mux.HandleFunc("DELETE /v1/memory/episodes/{id}", h.memoryEpisodeDelete)

	mux.HandleFunc("GET /v1/memory/facts", h.memoryFactsList)
	mux.HandleFunc("POST /v1/memory/facts", h.memoryFactCreate)
	mux.HandleFunc("PUT /v1/memory/facts/{id}", h.memoryFactUpdate)
	mux.HandleFunc("DELETE /v1/memory/facts/{id}", h.memoryFactDelete)

	// Legacy aliases (dashboard/cockpit may still call chunks).
	mux.HandleFunc("POST /v1/memory/chunks", h.memoryFactCreate)
	mux.HandleFunc("PUT /v1/memory/chunks/{id}", h.memoryFactUpdate)
	mux.HandleFunc("DELETE /v1/memory/chunks/{id}", h.memoryFactDelete)
}

type episodePayload struct {
	SessionID  string `json:"session_id"`
	Summary    string `json:"summary"`
	HappenedAt string `json:"happened_at"`
	Scope      string `json:"scope"`
}

type factPayload struct {
	Subject string `json:"subject"`
	Content string `json:"content"`
	Source  string `json:"source"`
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
	id, err := h.App.Episodic.CreateScoped(r.Context(), strings.TrimSpace(req.SessionID), userID, req.Scope, req.Summary, when)
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

func (h *Handler) memoryFactsList(w http.ResponseWriter, r *http.Request) {
	if h.App == nil || h.App.Facts == nil {
		writeError(w, http.StatusServiceUnavailable, "semantic memory not enabled (GEEGOO_PG_DSN)")
		return
	}
	userID := resolveUserID(r)
	limit := parseLimit(r, 50, 200)
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	var (
		rows []facts.Row
		err  error
	)
	if query != "" {
		rows, err = h.App.Facts.SearchRows(r.Context(), userID, query, limit)
	} else {
		rows, err = h.App.Facts.List(r.Context(), userID, limit)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]any{"facts": factRows(rows), "total": len(rows)})
}

func (h *Handler) memoryFactCreate(w http.ResponseWriter, r *http.Request) {
	if h.App == nil || h.App.Facts == nil {
		writeError(w, http.StatusServiceUnavailable, "semantic memory not enabled")
		return
	}
	var req factPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	content := strings.TrimSpace(req.Content)
	subject := strings.TrimSpace(req.Subject)
	if content == "" {
		writeError(w, http.StatusBadRequest, "content required")
		return
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "manual"
	}
	userID := resolveUserID(r)
	if err := h.App.Facts.Add(r.Context(), userID, subject, content, source); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	rows, _ := h.App.Facts.List(r.Context(), userID, 1)
	var row *facts.Row
	if len(rows) > 0 {
		row = &rows[0]
	}
	writeJSON(w, map[string]any{"ok": true, "fact": factRow(row), "chunk": factRow(row)})
}

func (h *Handler) memoryFactUpdate(w http.ResponseWriter, r *http.Request) {
	if h.App == nil || h.App.Facts == nil {
		writeError(w, http.StatusServiceUnavailable, "semantic memory not enabled")
		return
	}
	id, err := pathInt64(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid fact id")
		return
	}
	var req factPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	ok, err := h.App.Facts.Update(r.Context(), id, req.Content, req.Subject)
	if err != nil {
		writeFactErr(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "fact not found")
		return
	}
	row, _ := h.App.Facts.GetByID(r.Context(), id)
	writeJSON(w, map[string]any{"ok": true, "fact": factRow(row), "chunk": factRow(row)})
}

func (h *Handler) memoryFactDelete(w http.ResponseWriter, r *http.Request) {
	if h.App == nil || h.App.Facts == nil {
		writeError(w, http.StatusServiceUnavailable, "semantic memory not enabled")
		return
	}
	id, err := pathInt64(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid fact id")
		return
	}
	ok, err := h.App.Facts.Delete(r.Context(), id)
	if err != nil {
		writeFactErr(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "fact not found")
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
		"scope":       ep.Scope,
		"summary":     ep.Summary,
		"happened_at": ep.HappenedAt.Format(time.RFC3339),
		"created_at":  ep.CreatedAt.Format(time.RFC3339),
		"source":      "episodic",
	}
}

func factRows(rows []facts.Row) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for i := range rows {
		out = append(out, factRow(&rows[i]))
	}
	return out
}

func factRow(f *facts.Row) map[string]any {
	if f == nil {
		return map[string]any{}
	}
	raw := facts.Format(f.Subject, f.Content)
	return map[string]any{
		"id":         f.ID,
		"user_id":    f.UserID,
		"subject":    f.Subject,
		"content":    f.Content,
		"raw":        raw,
		"source":     f.Source,
		"created_at": f.CreatedAt.Format(time.RFC3339),
	}
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

func writeFactErr(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "fact not found")
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}
