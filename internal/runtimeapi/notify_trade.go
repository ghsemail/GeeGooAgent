package runtimeapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/feishupush"
	agentnotify "github.com/ghsemail/GeeGooAgent/internal/notify"
)

func (h *Handler) registerNotifyTradeRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/notify/trade", h.notifyTrade)
}

type notifyTradeReq struct {
	NoticeType int            `json:"notice_type"`
	Content    map[string]any `json:"content"`
}

func (h *Handler) notifyTrade(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireFeishuUser(w, r)
	if !ok {
		return
	}
	var req notifyTradeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	text := strings.TrimSpace(agentnotify.FormatTradeMarkdown(req.NoticeType, req.Content))
	if text == "" {
		writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": "empty notice"})
		return
	}

	sendCtx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	opts := feishupush.UserOpts{
		Workspace: h.feishuOutputDir(),
		UserID:    userID,
		Text:      text,
	}
	if h.App != nil {
		opts.DB = h.App.DB
		opts.PG = h.App.PG
	}
	if err := feishupush.SendUserWithRetry(sendCtx, opts); err != nil {
		slog.Warn("notify trade feishu failed", "user_id", userID, "notice_type", req.NoticeType, "err", err)
		writeJSONStatus(w, http.StatusBadGateway, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}
