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
	toolCtx.ClarifyFn = func(question string, choices []string) (string, bool) {
		if err := send(FormatClarifyMessage(question, choices)); err != nil {
			return "", false
		}
		if chatLock != nil {
			chatLock.Unlock()
		}
		raw, ok := hub.Wait(ctx, sessionKey)
		if chatLock != nil {
			chatLock.Lock()
		}
		if !ok {
			return "", false
		}
		answer, matched := ParseClarifyReply(raw, choices)
		if matched {
			return answer, true
		}
		if strings.TrimSpace(raw) != "" {
			return strings.TrimSpace(raw), true
		}
		return "", false
	}
	return toolCtx
}
