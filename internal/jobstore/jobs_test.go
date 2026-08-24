package jobstore_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/jobstore"
)

func TestRecordSkillVerdictUpdatesMatchingJob(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := jobstore.SaveJobs(dir, jobstore.DefaultJobs()); err != nil {
		t.Fatal(err)
	}
	if err := jobstore.RecordSkillVerdict(dir, "premarket_stock", "CN", "pass"); err != nil {
		t.Fatal(err)
	}
	jf, err := jobstore.LoadJobs(dir)
	if err != nil {
		t.Fatal(err)
	}
	var cn *jobstore.Job
	for i := range jf.Jobs {
		if jf.Jobs[i].Name == "premarket_stock_cn" {
			cn = &jf.Jobs[i]
			break
		}
	}
	if cn == nil {
		t.Fatal("premarket_stock_cn not found")
	}
	if cn.LastVerdict != "pass" {
		t.Fatalf("verdict=%q", cn.LastVerdict)
	}
	if cn.LastRun == "" {
		t.Fatal("expected last_run set")
	}
	for _, j := range jf.Jobs {
		if j.Name == "premarket_stock_hk" && j.LastVerdict == "pass" {
			t.Fatal("HK job should not be updated")
		}
	}
}

func TestRecordSkillVerdictNoOpForUnknownSkill(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := jobstore.SaveJobs(dir, jobstore.DefaultJobs()); err != nil {
		t.Fatal(err)
	}
	if err := jobstore.RecordSkillVerdict(dir, "intraday_stock", "CN", "pass"); err != nil {
		t.Fatal(err)
	}
	jf, err := jobstore.LoadJobs(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, j := range jf.Jobs {
		if j.LastVerdict != "" || j.LastRun != "" {
			t.Fatalf("unexpected update on %s", j.Name)
		}
	}
}
