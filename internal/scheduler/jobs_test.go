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
	if len(loaded.Jobs) != 9 {
		t.Fatalf("expected 9 default jobs, got %d: %+v", len(loaded.Jobs), loaded.Jobs)
	}
	skills := map[string]bool{}
	for _, j := range loaded.Jobs {
		skills[j.Skill] = true
	}
	for _, want := range []string{"premarket_market", "premarket_stock", "postmarket_stock"} {
		if !skills[want] {
			t.Fatalf("missing %s jobs: %+v", want, loaded.Jobs)
		}
	}
}

func TestDefaultJobsHasWeekdayPreMarket(t *testing.T) {
	t.Parallel()
	jf := scheduler.DefaultJobs()
	foundMarketCN := false
	foundStockCN := false
	foundPostCN := false
	foundPostHK := false
	foundPostUS := false
	for _, j := range jf.Jobs {
		if j.Skill == "premarket_market" && j.Market == "CN" && j.Enabled && j.Cron == "0 8 * * 1-5" {
			foundMarketCN = true
		}
		if j.Skill == "premarket_stock" && j.Market == "CN" && j.Enabled && j.Cron == "10 8 * * 1-5" {
			foundStockCN = true
		}
		if j.Skill == "postmarket_stock" && j.Market == "CN" && j.Enabled && j.Cron == "0 17 * * 1-5" {
			foundPostCN = true
		}
		if j.Skill == "postmarket_stock" && j.Market == "HK" && j.Enabled && j.Cron == "0 17 * * 1-5" {
			foundPostHK = true
		}
		if j.Skill == "postmarket_stock" && j.Market == "US" && j.Enabled && j.Cron == "0 5 * * 2-6" {
			foundPostUS = true
		}
	}
	if !foundMarketCN {
		t.Fatal("default jobs missing enabled premarket_market CN job")
	}
	if !foundStockCN {
		t.Fatal("default jobs missing enabled premarket_stock CN job")
	}
	if !foundPostCN || !foundPostHK {
		t.Fatal("default jobs missing enabled postmarket_stock CN/HK weekday jobs")
	}
	if !foundPostUS {
		t.Fatal("default jobs missing enabled postmarket_stock US job")
	}
}

func TestMigratePostmarketWeekdaySplit(t *testing.T) {
	t.Parallel()
	jf := &scheduler.JobsFile{
		Jobs: []scheduler.Job{
			{Name: "postmarket_stock_weekday", Skill: "postmarket_stock", Cron: "0 17 * * 1-5", Enabled: true},
		},
	}
	if !scheduler.MigrateJobs(jf) {
		t.Fatal("expected migration")
	}
	enabled := map[string]string{}
	for _, j := range jf.Jobs {
		if j.Skill == "postmarket_stock" && j.Enabled {
			enabled[j.Market] = j.Cron
		}
	}
	if enabled["CN"] != "0 17 * * 1-5" || enabled["HK"] != "0 17 * * 1-5" {
		t.Fatalf("cn/hk postmarket jobs: %+v", enabled)
	}
	if enabled["US"] != "0 5 * * 2-6" {
		t.Fatalf("us postmarket job: %+v", enabled)
	}
	for _, j := range jf.Jobs {
		if j.Name == "postmarket_stock_weekday" {
			t.Fatal("legacy weekday job should be removed")
		}
	}
	if len(jf.Jobs) != 3 {
		t.Fatalf("expected 3 postmarket jobs after migration, got %d: %+v", len(jf.Jobs), jf.Jobs)
	}
}

func TestPruneLegacyPostmarketWeekday(t *testing.T) {
	t.Parallel()
	jf := &scheduler.JobsFile{
		Jobs: []scheduler.Job{
			{Name: "postmarket_stock_weekday", Skill: "postmarket_stock", Cron: "0 17 * * 1-5", Enabled: false},
			{Name: "postmarket_stock_cn", Skill: "postmarket_stock", Market: "CN", Cron: "0 17 * * 1-5", Enabled: true},
		},
	}
	if !scheduler.MigrateJobs(jf) {
		t.Fatal("expected prune")
	}
	if len(jf.Jobs) != 3 {
		t.Fatalf("expected cn/hk/us postmarket jobs, got %d: %+v", len(jf.Jobs), jf.Jobs)
	}
	for _, j := range jf.Jobs {
		if j.Name == "postmarket_stock_weekday" {
			t.Fatal("legacy weekday job should be removed")
		}
	}
}

func TestMigrateJobsLegacySkillNames(t *testing.T) {
	t.Parallel()
	jf := &scheduler.JobsFile{
		Jobs: []scheduler.Job{
			{Name: "pre_market_cn", Skill: "pre_market", Cron: "0 8 * * 1-5", Enabled: true},
			{Name: "pre_market_stock_hk", Skill: "pre_market_stock", Cron: "10 9 * * 1-5", Enabled: true},
		},
	}
	if !scheduler.MigrateJobs(jf) {
		t.Fatal("expected migration")
	}
	if jf.Jobs[0].Skill != "premarket_market" || jf.Jobs[0].Market != "CN" {
		t.Fatalf("cn market job: %+v", jf.Jobs[0])
	}
	if jf.Jobs[1].Skill != "premarket_stock" || jf.Jobs[1].Market != "HK" {
		t.Fatalf("hk stock job: %+v", jf.Jobs[1])
	}
}

func TestFormatJobRendersState(t *testing.T) {
	t.Parallel()
	j := scheduler.Job{Name: "j1", Skill: "premarket_market", Market: "CN", Cron: "0 8 * * 1-5", Enabled: true,
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
	custom := `{"version":1,"jobs":[{"name":"x","skill":"premarket_stock","market":"CN","cron":"*/5 * * * *","enabled":true,"platform":"log"}]}`
	if err := os.WriteFile(path, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}
	jf, err := scheduler.LoadJobs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if jf.Jobs[0].Cron != "*/5 * * * *" || jf.Jobs[0].Market != "CN" {
		t.Fatalf("round-trip failed: %+v", jf.Jobs[0])
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
