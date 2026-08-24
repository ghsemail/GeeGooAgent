// Package scheduler runs skill jobs on a cron schedule inside the agent
// process, replacing the external systemd timer for production. On a run
// whose supervisor verdict is recoverable or terminal, a one-shot retry is
// scheduled after a backoff delay.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
	"github.com/robfig/cron/v3"
)

// Runner is the long-running scheduler.
type Runner struct {
	app       *app.App
	jobsDir   string
	cron      *cron.Cron
	mu          sync.Mutex
	running     sync.Map // job name -> struct{}; skip overlapping cron ticks
	retryIn     time.Duration // backoff for recoverable retries
	maxRetries  int
	retryCounts map[string]int
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
		retryIn: 30 * time.Minute, maxRetries: 2, retryCounts: map[string]int{},
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
// retry when the verdict is recoverable or terminal (up to maxRetries).
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
		time.AfterFunc(5*time.Minute, func() { r.executeAndMaybeRetry(jobRef) })
		return
	}
	result, err := application.RunSkillContext(context.Background(), job.Skill, app.SkillRunOptions{
		Market:       job.Market,
		NotifyFeishu: strings.EqualFold(strings.TrimSpace(job.Platform), "feishu") && shouldNotifyJob(job),
	})
	verdict := "unknown"
	if result.Supervisor != nil {
		verdict = string(result.Supervisor.Verdict)
	}
	if err != nil {
		verdict = "error"
	}
	r.recordRun(job, verdict)
	if verdict == "pass" {
		r.maybeNotifyFeishu(job, result)
		return
	}
	// Schedule a retry with backoff if under the retry cap.
	r.mu.Lock()
	count := r.retryCounts[job.Name]
	if count < r.maxRetries {
		r.retryCounts[job.Name] = count + 1
	}
	r.mu.Unlock()
	if count >= r.maxRetries {
		r.maybeNotifyFeishu(job, result)
		return
	}
	delay := r.retryIn * time.Duration(1<<count) // 30m, 60m
	jobRef := job
	time.AfterFunc(delay, func() { r.executeAndMaybeRetry(jobRef) })
}

func (r *Runner) recordRun(job Job, verdict string) {
	jf, err := LoadJobs(r.jobsDir)
	if err != nil || jf == nil {
		return
	}
	for i := range jf.Jobs {
		if jf.Jobs[i].Name == job.Name {
			jf.Jobs[i].LastRun = time.Now().UTC().Format(time.RFC3339)
			jf.Jobs[i].LastVerdict = verdict
			break
		}
	}
	_ = SaveJobs(r.jobsDir, jf)
}

// VerdictForTest exposes the retry-count logic boundary for tests.
func VerdictForTest(report *workflow.SupervisorReport) string {
	if report == nil {
		return "unknown"
	}
	return string(report.Verdict)
}
