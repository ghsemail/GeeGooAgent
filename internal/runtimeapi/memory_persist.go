package runtimeapi

import (
	"context"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
)

type turnMemoryResult struct {
	SummaryStored bool
	EpisodeStored bool
	SummaryChars  int
}

// persistTurnMemory writes per-turn semantic summary + light episodic snapshot.
func (h *Handler) persistTurnMemory(ctx context.Context, chat *chatsession.ChatSession, userID string) turnMemoryResult {
	var res turnMemoryResult
	if h == nil || h.App == nil || chat == nil {
		return res
	}
	summary := agent.CleanAssistantVisibleText(chat.Summary)
	if summary == "" {
		return res
	}
	res.SummaryChars = len(summary)
	uid := strings.TrimSpace(userID)
	if uid == "" {
		uid = chatsession.UserIDFromSession(chat)
	}
	if h.App.Semantic != nil {
		if err := h.App.Semantic.UpsertSummary(ctx, chat.ID, uid, summary); err == nil {
			res.SummaryStored = true
		}
	}
	if h.App.Episodic != nil {
		if err := h.App.Episodic.Add(ctx, chat.ID, uid, summary, time.Now().UTC()); err == nil {
			res.EpisodeStored = true
		}
	}
	return res
}
