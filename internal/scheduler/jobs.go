package scheduler

import "github.com/ghsemail/GeeGooAgent/internal/jobstore"

type (
	// Job is one scheduled agent task.
	Job = jobstore.Job
	// JobsFile is the persisted job list.
	JobsFile = jobstore.JobsFile
)

func LoadJobs(dir string) (*JobsFile, error) {
	return jobstore.LoadJobs(dir)
}

func SaveJobs(dir string, jf *JobsFile) error {
	return jobstore.SaveJobs(dir, jf)
}

func DefaultJobs() *JobsFile {
	return jobstore.DefaultJobs()
}

func MigrateJobs(jf *JobsFile) bool {
	return jobstore.MigrateJobs(jf)
}

func SortJobs(jf *JobsFile) {
	jobstore.SortJobs(jf)
}

func FormatJob(j Job) string {
	return jobstore.FormatJob(j)
}
