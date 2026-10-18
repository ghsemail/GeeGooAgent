package signalclient

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// MockClient is an in-memory lab-api for unit tests.
type MockClient struct {
	mu      sync.Mutex
	seq     int
	jobs    map[string]*Job
	BarsFn  func(req KlineJobRequest) ([]any, error)
	Instant bool // when true, jobs complete immediately on create
}

// NewMockClient returns a mock with empty job store.
func NewMockClient() *MockClient {
	return &MockClient{jobs: map[string]*Job{}, Instant: true}
}

func (m *MockClient) CreateKlineJob(ctx context.Context, req KlineJobRequest) (string, error) {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	id := fmt.Sprintf("job-%d", m.seq)
	job := &Job{ID: id, Status: JobPending}
	if m.Instant {
		bars, err := m.resolveBars(req)
		if err != nil {
			job.Status = JobFailed
			job.Error = err.Error()
		} else {
			job.Status = JobCompleted
			job.Result = &KlineJobResult{Bars: bars}
		}
	}
	m.jobs[id] = job
	return id, nil
}

func (m *MockClient) GetJob(ctx context.Context, jobID string) (Job, error) {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[strings.TrimSpace(jobID)]
	if !ok {
		return Job{}, fmt.Errorf("lab-api mock: job not found")
	}
	return *job, nil
}

func (m *MockClient) resolveBars(req KlineJobRequest) ([]any, error) {
	if m.BarsFn != nil {
		return m.BarsFn(req)
	}
	return nil, nil
}
