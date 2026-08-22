package runtimeapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/clients/weknora"
	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func (h *Handler) registerKnowledgeRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/knowledge/overview", h.knowledgeOverview)
	mux.HandleFunc("GET /v1/knowledge/tree", h.knowledgeTree)
	mux.HandleFunc("GET /v1/knowledge/documents", h.knowledgeDocuments)
}

func (h *Handler) knowledgeOverview(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, h.collectKnowledgeOverview(r.Context()))
}

func (h *Handler) knowledgeTree(w http.ResponseWriter, r *http.Request) {
	cfg := h.appConfig().ResolvedWeKnora()
	client := h.weknoraClient()
	out := map[string]any{
		"ok":      false,
		"web_url": cfg.WebURL,
		"kb_id":   cfg.KBID,
		"tree":    weknora.FolderTree{},
	}
	if client == nil || !client.Configured() {
		out["error"] = "weknora is not configured"
		writeJSON(w, out)
		return
	}
	tree, err := client.Folders(r.Context())
	if err != nil {
		out["error"] = err.Error()
		writeJSON(w, out)
		return
	}
	out["ok"] = true
	out["tree"] = tree
	writeJSON(w, out)
}

func (h *Handler) knowledgeDocuments(w http.ResponseWriter, r *http.Request) {
	cfg := h.appConfig().ResolvedWeKnora()
	client := h.weknoraClient()
	q := r.URL.Query()
	_, filterFolder := q["folder_path"]
	out := map[string]any{
		"ok":        false,
		"web_url":   cfg.WebURL,
		"kb_id":     cfg.KBID,
		"documents": []any{},
	}
	if filterFolder {
		out["folder_path"] = q.Get("folder_path")
	}
	if client == nil || !client.Configured() {
		out["error"] = "weknora is not configured"
		writeJSON(w, out)
		return
	}
	docs, err := client.ListDocuments(r.Context(), weknora.ListDocumentsOpts{
		FolderPath:   q.Get("folder_path"),
		FilterFolder: filterFolder,
	})
	if err != nil {
		out["error"] = err.Error()
		writeJSON(w, out)
		return
	}
	rows := make([]map[string]any, 0, len(docs))
	for _, d := range docs {
		rows = append(rows, map[string]any{
			"id":           d.ID,
			"file_name":    d.FileName,
			"title":        d.Title,
			"folder_path":  d.FolderPath,
			"file_size":    d.FileSize,
			"parse_status": d.ParseStatus,
			"updated_at":   d.UpdatedAt,
		})
	}
	out["ok"] = true
	out["documents"] = rows
	writeJSON(w, out)
}

func (h *Handler) collectKnowledgeOverview(ctx context.Context) map[string]any {
	cfg := h.appConfig().ResolvedWeKnora()
	out := map[string]any{
		"ok":      false,
		"web_url": cfg.WebURL,
		"kb_id":   cfg.KBID,
	}
	client := h.weknoraClient()
	if client == nil || !client.Configured() {
		out["error"] = "weknora is not configured"
		return out
	}
	if err := client.Health(ctx); err != nil {
		out["error"] = err.Error()
		return out
	}
	kb, err := client.GetKnowledgeBase(ctx)
	if err != nil {
		out["error"] = err.Error()
		return out
	}
	docs, err := client.ListDocuments(ctx, weknora.ListDocumentsOpts{})
	if err != nil {
		out["error"] = err.Error()
		return out
	}
	parsing, failed := 0, 0
	for _, d := range docs {
		switch strings.ToLower(d.ParseStatus) {
		case "pending", "processing", "finalizing":
			parsing++
		case "failed":
			failed++
		}
	}
	chatName, embName := "", ""
	if kb.ChatModelID != "" {
		if m, err := client.GetModel(ctx, kb.ChatModelID); err == nil {
			chatName = m.Name
		}
	}
	if kb.EmbeddingModelID != "" {
		if m, err := client.GetModel(ctx, kb.EmbeddingModelID); err == nil {
			embName = m.Name
		}
	}
	out["ok"] = true
	out["kb_name"] = kb.Name
	out["kb_id"] = kb.ID
	out["description"] = kb.Description
	out["chat_model"] = chatName
	out["embedding_model"] = embName
	out["document_count"] = len(docs)
	out["parsing_count"] = parsing
	out["failed_count"] = failed
	return out
}

func (h *Handler) weknoraClient() *weknora.Client {
	if h != nil && h.App != nil && h.App.WeKnora != nil {
		return h.App.WeKnora
	}
	if h == nil {
		return nil
	}
	cfg := h.appConfig()
	if cfg == nil {
		cfg = &config.AppConfig{}
	}
	return weknora.NewFromResolved(cfg.ResolvedWeKnora())
}
