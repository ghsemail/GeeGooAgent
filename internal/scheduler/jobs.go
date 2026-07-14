package scheduler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Job is one scheduled agent task.
type Job struct {
	Name     string `json:"name"`
	Skill    string `json:"skill"`
	Cron     string `json:"cron"`      // standard 5-field cron: "0 8 * * 1-5"
	Prompt   string `json:"prompt"`    // optional prompt context injected into the run
	Enabled  bool   `json:"enabled"`
	Platform string `json:"platform"`  // reserved: where to deliver result (log only for now)
	LastRun  string `json:"last_run"`  // RFC3339, updated after each tick
	LastVerdict string `json:"last_verdict"`
}

// JobsFile is the persisted job list.
type JobsFile struct {
	Version int   `json:"version"`
	Jobs    []Job `json:"jobs"`
}

// LoadJobs reads jobs.json from dir. Returns an empty list if the file does
// not exist (callers may then seed defaults).
func LoadJobs(dir string) (*JobsFile, error) {
	path := filepath.Join(dir, "jobs.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &JobsFile{Version: 1}, nil
		}
		return nil, err
	}
	var jf JobsFile
	if err := json.Unmarshal(raw, &jf); err != nil {
		return nil, fmt.Errorf("parse jobs.json: %w", err)
	}
	if jf.Version == 0 {
		jf.Version = 1
	}
	return &jf, nil
}

// SaveJobs writes jobs.json atomically-ish to dir.
func SaveJobs(dir string, jf *JobsFile) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	jf.Version = 1
	raw, err := json.MarshalIndent(jf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "jobs.json"), append(raw, '\n'), 0o644)
}

// DefaultJobs returns a sensible default job set (pre_market weekdays 08:00).
func DefaultJobs() *JobsFile {
	return &JobsFile{
		Version: 1,
		Jobs: []Job{
			{Name: "pre_market_weekday", Skill: "pre_market", Cron: "0 8 * * 1-5",
				Enabled: true, Platform: "log"},
		},
	}
}

// SortJobs orders jobs by name for stable display.
func SortJobs(jf *JobsFile) {
	sort.Slice(jf.Jobs, func(i, j int) bool { return jf.Jobs[i].Name < jf.Jobs[j].Name })
}

// FormatJob renders one job for `geegoo scheduler list`.
func FormatJob(j Job) string {
	state := "disabled"
	if j.Enabled {
		state = "enabled"
	}
	last := j.LastRun
	if last == "" {
		last = "(never)"
	} else if t, err := time.Parse(time.RFC3339, last); err == nil {
		last = t.Local().Format("2006-01-02 15:04")
	}
	verdict := j.LastVerdict
	if verdict == "" {
		verdict = "-"
	}
	return fmt.Sprintf("%-22s  %-12s  %-14s  %-8s  verdict=%-10s  last=%s",
		j.Name, j.Skill, j.Cron, state, verdict, last)
}
