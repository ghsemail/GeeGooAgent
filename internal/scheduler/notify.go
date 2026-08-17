package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/notify"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func (r *Runner) maybeNotifyFeishu(job Job, result workflow.RunResult) {
	if r == nil || r.app == nil || !shouldNotifyJob(job) {
		return
	}
	text := BuildFeishuSummary(job, result)
	if text == "" {
		return
	}
	webhook := ""
	if r.app.Config != nil {
		webhook = r.app.Config.EffectiveFeishuWebhookURL()
	}
	sender := notify.NewFeishuSender(webhook)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if err := sender.Send(ctx, text); err != nil {
			lastErr = err
			slog.Warn("scheduler: feishu notify failed", "job", job.Name, "attempt", attempt+1, "err", err)
			time.Sleep(10 * time.Second)
			continue
		}
		slog.Info("scheduler: feishu summary sent", "job", job.Name, "chars", len(text))
		return
	}
	if lastErr != nil {
		slog.Error("scheduler: feishu notify gave up", "job", job.Name, "err", lastErr)
	}
}
