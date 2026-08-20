package scheduler

import "testing"

func TestRunJobSkipsOverlap(t *testing.T) {
	t.Parallel()
	r := &Runner{retryCounts: map[string]int{}}
	job := Job{Name: "premarket_stock_us", Skill: "premarket_stock", Market: "US"}
	r.running.Store(job.Name, struct{}{})
	r.runJob(job)
	if _, ok := r.running.Load(job.Name); !ok {
		t.Fatal("expected overlap skip to leave running marker intact")
	}
}
