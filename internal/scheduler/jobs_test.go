package scheduler_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/scheduler"
)

func TestLoadJobsMissingReturnsEmpty(t *testing.T) {
	t.Parallel()
	jf, err := scheduler.LoadJobs(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(jf.Jobs) != 0 {
		t.Fatalf("expected empty, got %d", len(jf.Jobs))
	}
}

func TestSaveAndReloadJobs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	jf := scheduler.DefaultJobs()
	if err := scheduler.SaveJobs(dir, jf); err != nil {
		t.Fatal(err)
	}
	loaded, err := scheduler.LoadJobs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Jobs) != 1 || loaded.Jobs[0].Skill != "pre_market" {
		t.Fatalf("unexpected: %+v", loaded)
	}
}

func TestDefaultJobsHasWeekdayPreMarket(t *testing.T) {
	t.Parallel()
	jf := scheduler.DefaultJobs()
	found := false
	for _, j := range jf.Jobs {
		if j.Skill == "pre_market" && j.Enabled && j.Cron == "0 8 * * 1-5" {
			found = true
		}
	}
	if !found {
		t.Fatal("default jobs missing enabled pre_market weekday job")
	}
}

func TestFormatJobRendersState(t *testing.T) {
	t.Parallel()
	j := scheduler.Job{Name: "j1", Skill: "pre_market", Cron: "0 8 * * 1-5", Enabled: true,
		LastRun: time.Now().UTC().Format(time.RFC3339), LastVerdict: "pass"}
	s := scheduler.FormatJob(j)
	if !contains(s, "enabled") || !contains(s, "verdict=pass") {
		t.Fatalf("format missing fields: %s", s)
	}
}

func TestJobsFileRoundTripsArbitraryFields(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "jobs.json")
	custom := `{"version":1,"jobs":[{"name":"x","skill":"pre_market","cron":"*/5 * * * *","enabled":true,"platform":"log"}]}`
	if err := os.WriteFile(path, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}
	jf, err := scheduler.LoadJobs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if jf.Jobs[0].Cron != "*/5 * * * *" {
		t.Fatalf("cron round-trip failed: %s", jf.Jobs[0].Cron)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
