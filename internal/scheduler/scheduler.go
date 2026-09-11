// Package scheduler runs skill jobs on a cron schedule inside the agent
// process, replacing the external systemd timer for production. When report
// generation and Feishu delivery do not succeed, the job is retried every
// retryIn up to maxRetries times.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/jobstore"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
	"github.com/robfig/cron/v3"
)

const (
	defaultJobRetryInterval = 5 * time.Minute
	defaultJobMaxRetries    = 3
)

// Runner is the long-running scheduler.
type Runner struct {
	app         *app.App
	jobsDir     string
	cron        *cron.Cron
	mu          sync.Mutex
	running     sync.Map // job name -> struct{}; skip overlapping cron ticks
	retryIn     time.Duration
	maxRetries  int
	retryCounts map[string]int
	sleep       func(time.Duration)
}

// NewRunner creates a scheduler backed by the given app and jobs directory.
func NewRunner(application *app.App, jobsDir string) *Runner {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	return &Runner{
		app: application, jobsDir: jobsDir,
		cron: cron.New(cron.WithLocation(loc)),
		retryIn: defaultJobRetryInterval, maxRetries: defaultJobMaxRetries,
		retryCounts: map[string]int{},
		sleep:       time.Sleep,
	}
}

// Start loads jobs.json, registers enabled jobs, and starts the cron loop.
// Returns when ctx is cancelled or the cron stops.
func (r *Runner) Start(ctx context.Context) error {
	jf, err := LoadJobs(r.jobsDir)
	if err != nil {
		return fmt.Errorf("load jobs: %w", err)
	}
	if len(jf.Jobs) == 0 {
		jf = DefaultJobs()
		_ = SaveJobs(r.jobsDir, jf)
	} else if MigrateJobs(jf) {
		_ = SaveJobs(r.jobsDir, jf)
	}
	for i := range jf.Jobs {
		job := jf.Jobs[i]
		if !job.Enabled {
			continue
		}
		jobRef := job
		if _, err := r.cron.AddFunc(job.Cron, func() { r.runJob(jobRef) }); err != nil {
			return fmt.Errorf("register job %s (%s): %w", job.Name, job.Cron, err)
		}
	}
	r.cron.Start()
	<-ctx.Done()
	stopCtx := r.cron.Stop()
	<-stopCtx.Done()
	return ctx.Err()
}

// runJob executes one skill, records the supervisor verdict, and schedules a
// retry when report delivery fails (up to maxRetries).
func (r *Runner) runJob(job Job) {
	if _, loaded := r.running.LoadOrStore(job.Name, struct{}{}); loaded {
		slog.Info("scheduler: job already running, skip", "job", job.Name)
		return
	}
	defer r.running.Delete(job.Name)

	r.mu.Lock()
	r.retryCounts[job.Name] = 0
	r.mu.Unlock()
	r.executeAndMaybeRetry(job)
}

func (r *Runner) executeAndMaybeRetry(job Job) {
	application := r.app
	if application == nil {
		return
	}
	if skillRequiresSynthesis(job.Skill) && !application.SynthesisReady() {
		slog.Error("scheduler: synthesis not ready, deferring job", "job", job.Name, "skill", job.Skill, "market", job.Market)
		r.recordRun(job, "deferred")
		jobRef := job
		time.AfterFunc(defaultJobRetryInterval, func() { r.executeAndMaybeRetry(jobRef) })
		return
	}
	notifyFeishu := strings.EqualFold(strings.TrimSpace(job.Platform), "feishu") && shouldNotifyJob(job)
	result, err := application.RunSkillContext(context.Background(), job.Skill, app.SkillRunOptions{
		Market:       job.Market,
		NotifyFeishu: notifyFeishu,
	})
	delivered, failReason := app.SkillDeliveryReasonForScheduler(result, err)
	verdict := verdictForRun(result, err, delivered)
	r.recordRun(job, verdict)
	if delivered {
		r.maybeNotifyFeishu(job, result)
		return
	}

	r.mu.Lock()
	count := r.retryCounts[job.Name]
	if count < r.maxRetries {
		r.retryCounts[job.Name] = count + 1
	}
	r.mu.Unlock()
	if count >= r.maxRetries {
		slog.Error("scheduler: report delivery gave up after retries",
			"job", job.Name,
			"skill", job.Skill,
			"market", job.Market,
			"retry_count", count,
			"max_retries", r.maxRetries,
			"reason", failReason,
		)
		return
	}
	slog.Warn("scheduler: report delivery failed, will retry",
		"job", job.Name,
		"skill", job.Skill,
		"market", job.Market,
		"retry", count+1,
		"max_retries", r.maxRetries,
		"retry_in", r.retryIn.String(),
		"reason", failReason,
	)
	jobRef := job
	retryNum := count + 1
	time.AfterFunc(r.retryIn, func() {
		slog.Info("scheduler: report delivery retrying",
			"job", jobRef.Name,
			"retry", retryNum,
			"max_retries", r.maxRetries,
		)
		r.executeAndMaybeRetry(jobRef)
	})
}

func verdictForRun(result workflow.RunResult, err error, delivered bool) string {
	if delivered {
		return "pass"
	}
	if err != nil {
		return "error"
	}
	if result.Supervisor != nil {
		return string(result.Supervisor.Verdict)
	}
	return "delivery_failed"
}

func (r *Runner) recordRun(job Job, verdict string) {
	_ = jobstore.RecordSkillVerdict(r.jobsDir, job.Skill, job.Market, verdict)
}

// JobRetryPolicy returns report-delivery retry settings for scheduled jobs.
func JobRetryPolicy() (interval time.Duration, maxRetries int) {
	return defaultJobRetryInterval, defaultJobMaxRetries
}

// VerdictForTest exposes the retry-count logic boundary for tests.
func VerdictForTest(report *workflow.SupervisorReport) string {
	if report == nil {
		return "unknown"
	}
	return string(report.Verdict)
}
