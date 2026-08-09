// M2+ reserved surface (stubs). Full implementations land in follow-up PRs.
package gateway

import "context"

// DeliverToHome sends text to the configured home channel for a platform (M2).
func (r *Runner) DeliverToHome(ctx context.Context, platform Platform, text string) error {
	_ = ctx
	_ = platform
	_ = text
	return ErrNotImplemented{Feature: "DeliverToHome (M2)"}
}

// NotifySchedulerResult hooks Job.Platform=feishu delivery (M2).
func (r *Runner) NotifySchedulerResult(ctx context.Context, platform Platform, jobName, summary string) error {
	_ = ctx
	_ = platform
	_ = jobName
	_ = summary
	return ErrNotImplemented{Feature: "NotifySchedulerResult (M2)"}
}
