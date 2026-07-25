package runtimeapi

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
)

type turnMemoryResult struct {
	SummaryStored bool
	EpisodeStored bool
	SummaryChars  int
}

// persistTurnMemory is intentionally a no-op: Waku writes facts/episodes only via consolidation.
func (h *Handler) persistTurnMemory(_ context.Context, _ *chatsession.ChatSession, _ string) turnMemoryResult {
	return turnMemoryResult{}
}
