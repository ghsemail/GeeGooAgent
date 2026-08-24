package gateway

import (
	"context"
	"strings"
	"sync"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func WireIMClarify(
	ctx context.Context,
	toolCtx tools.Context,
	hub *ClarifyHub,
	sessionKey string,
	chatLock *sync.Mutex,
	send func(text string) error,
) tools.Context {
	if hub == nil || send == nil {
		return toolCtx
	}
	toolCtx.ClarifyFn = func(waitCtx context.Context, question string, choices []string) (string, bool) {
		if err := send(FormatClarifyMessage(question, choices)); err != nil {
			return "", false
		}
		return waitIMClarifyReply(waitCtx, ctx, hub, sessionKey, chatLock, send, choices)
	}
	return toolCtx
}

func waitIMClarifyReply(
	waitCtx context.Context,
	fallback context.Context,
	hub *ClarifyHub,
	sessionKey string,
	chatLock *sync.Mutex,
	send func(text string) error,
	choices []string,
) (string, bool) {
	useCtx := waitCtx
	if useCtx == nil {
		useCtx = fallback
	}
	for {
		if chatLock != nil {
			chatLock.Unlock()
		}
		raw, ok := hub.Wait(useCtx, sessionKey)
		if chatLock != nil {
			chatLock.Lock()
		}
		if !ok {
			return "", false
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if IsClarifySkip(raw) {
			return "", false
		}
		answer, matched := ParseClarifyReply(raw, choices)
		if matched {
			return answer, true
		}
		if len(choices) > 0 && IsOtherClarifySelection(raw, choices) {
			if err := send(FormatClarifyCustomPrompt()); err != nil {
				return "", false
			}
			continue
		}
		if raw != "" {
			return raw, true
		}
	}
}
