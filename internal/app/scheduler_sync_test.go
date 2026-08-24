package app

import (
	"errors"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/jobstore"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func TestSchedulerVerdictFromResult(t *testing.T) {
	t.Parallel()
	pass := workflow.RunResult{Status: "completed", Supervisor: &workflow.SupervisorReport{Verdict: workflow.VerdictPass}}
	if got := schedulerVerdictFromResult(pass, nil); got != "pass" {
		t.Fatalf("got %q", got)
	}
	term := workflow.RunResult{Status: "failed", Supervisor: &workflow.SupervisorReport{Verdict: workflow.VerdictTerminal}}
	if got := schedulerVerdictFromResult(term, nil); got != "terminal" {
		t.Fatalf("got %q", got)
	}
	if got := schedulerVerdictFromResult(workflow.RunResult{}, errors.New("boom")); got != "error" {
		t.Fatalf("got %q", got)
	}
}

func TestSyncScheduledJobVerdictPersists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	a := &App{Workspace: dir}
	if err := jobstore.SaveJobs(dir+"/scheduler", jobstore.DefaultJobs()); err != nil {
		t.Fatal(err)
	}
	result := workflow.RunResult{
		Status:     "completed",
		Supervisor: &workflow.SupervisorReport{Verdict: workflow.VerdictPass},
	}
	a.syncScheduledJobVerdict("premarket_stock", "HK", result, nil)
	jf, err := jobstore.LoadJobs(dir + "/scheduler")
	if err != nil {
		t.Fatal(err)
	}
	for _, j := range jf.Jobs {
		if j.Name == "premarket_stock_hk" {
			if j.LastVerdict != "pass" {
				t.Fatalf("verdict=%q", j.LastVerdict)
			}
			return
		}
	}
	t.Fatal("premarket_stock_hk not found")
}
