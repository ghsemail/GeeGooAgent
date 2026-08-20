package app

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/feishupush"
	"github.com/ghsemail/GeeGooAgent/internal/opslog"
	"github.com/ghsemail/GeeGooAgent/internal/stockdigest"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func (a *App) maybeNotifyUserStockFeishu(ctx context.Context, userID, skill, market string, result workflow.RunResult) bool {
	if a == nil {
		return false
	}
	if reason := stockdigest.NotifySkipReason(skill, market, result); reason != "" {
		slog.Info("app: stock feishu notify skipped", "user_id", userID, "skill", skill, "market", market, "reason", reason)
		return false
	}
	text := stockdigest.Build(skill, market, result)
	if strings.TrimSpace(text) == "" || stockdigest.IsEmptyDigestMessage(text) {
		slog.Info("app: stock feishu notify skipped", "user_id", userID, "skill", skill, "market", market, "reason", "empty_digest")
		return false
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
		return false
	}
	return true
}

func (a *App) persistReportGenerationLog(
	ctx context.Context,
	rec *opslog.RunRecorder,
	result workflow.RunResult,
	feishuSent bool,
	feishuSkipReason string,
) {
	if a == nil || rec == nil {
		return
	}
	opslog.PersistFromResult(ctx, a.Config, rec, result, feishuSent, feishuSkipReason)
}
