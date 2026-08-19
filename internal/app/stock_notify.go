package app

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/feishupush"
	"github.com/ghsemail/GeeGooAgent/internal/stockdigest"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func (a *App) maybeNotifyUserStockFeishu(ctx context.Context, userID, skill, market string, result workflow.RunResult) {
	if a == nil || !result.OK() {
		return
	}
	if result.Supervisor != nil && result.Supervisor.Verdict != workflow.VerdictPass {
		return
	}
	text := stockdigest.Build(skill, market, result)
	if strings.TrimSpace(text) == "" {
		return
	}
	sendCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if err := feishupush.SendUserWithRetry(sendCtx, feishupush.UserOpts{
		Config:    a.Config,
		Workspace: a.feishuStoreDir(),
		UserID:    userID,
		Text:      text,
		DB:        a.DB,
		PG:        a.PG,
	}); err != nil {
		slog.Warn("app: stock feishu notify failed", "user_id", userID, "skill", skill, "market", market, "err", err)
	}
}

func (a *App) feishuStoreDir() string {
	if a == nil {
		return "."
	}
	if a.Config != nil {
		if d, err := a.Config.ResolveOutputDir(); err == nil && strings.TrimSpace(d) != "" {
			return d
		}
	}
	if strings.TrimSpace(a.Workspace) != "" {
		return a.Workspace
	}
	return "."
}
