package signalclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client talks to GeeGooSignal lab-api on :3250.
// Task lifecycle is always queried via lab-api (GET /v1/lab/jobs/{job_id}); Agent never calls lab-worker.
// v1 has no batch endpoint — callers create one job per timeframe.
type Client interface {
	CreateKlineJob(ctx context.Context, req KlineJobRequest) (jobID string, err error)
	GetJob(ctx context.Context, jobID string) (Job, error)
}

// KlineJobRequest asks lab-api to fetch OHLCV (取数窗).
// When AsOf is set, lab-api returns only bars with time <= as_of (Asia/Shanghai, same as probe).
// When AsOf is nil, bars run through latest available.
type KlineJobRequest struct {
	Code       string     `json:"code"`
	Frequency  string     `json:"frequency"`
	MonthsBack int        `json:"months_back"`
	AsOf       *time.Time `json:"as_of,omitempty"`
}

// Job is a lab-api async task snapshot from GET /v1/lab/jobs/{job_id} (HTTP 200 while polling).
type Job struct {
	ID     string          `json:"id"`
	Status JobStatus       `json:"status"`
	Error  string          `json:"error,omitempty"`
	Result *KlineJobResult `json:"result,omitempty"`
}

// JobStatus is the finalized lab-api task enum.
type JobStatus string

const (
	JobPending   JobStatus = "pending"
	JobRunning   JobStatus = "running"
	JobCompleted JobStatus = "completed"
	JobFailed    JobStatus = "failed"
)

// KlineJobResult is required when status=completed (result.bars must be present, may be empty).
type KlineJobResult struct {
	Bars []any `json:"bars"`
}

// APIError is returned for lab-api HTTP 4xx/5xx with JSON {"error":"..."}.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e == nil {
		return "lab-api error"
	}
	if e.Message != "" {
		return fmt.Sprintf("lab-api: HTTP %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("lab-api: HTTP %d", e.StatusCode)
}

// HTTPClient implements Client against GeeGooSignal lab-api.
// Signal lab-worker runs at most 4 jobs concurrently; extra jobs stay pending — polling handles queueing.
type HTTPClient struct {
	BaseURL    string
	APIKey     string
	HTTP       *http.Client
	PollEvery  time.Duration
	PollBudget time.Duration
	// FetchAttempts is whole fetch retries (new job) after retryable errors; default 3.
	FetchAttempts int
	RetryBase     time.Duration
}

// NewHTTPClient returns a client with sensible poll defaults.
func NewHTTPClient(baseURL, apiKey string) *HTTPClient {
	return &HTTPClient{
		BaseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		APIKey:     strings.TrimSpace(apiKey),
		HTTP:       &http.Client{Timeout: 60 * time.Second},
		PollEvery:  200 * time.Millisecond,
		PollBudget: 45 * time.Second,
	}
}

const asOfLayout = "2006-01-02 15:04:05"

func (c *HTTPClient) CreateKlineJob(ctx context.Context, req KlineJobRequest) (string, error) {
	body := map[string]any{
		"code":        strings.TrimSpace(req.Code),
		"frequency":   strings.TrimSpace(req.Frequency),
		"months_back": req.MonthsBack,
	}
	if req.AsOf != nil && !req.AsOf.IsZero() {
		loc, _ := time.LoadLocation("Asia/Shanghai")
		body["as_of"] = req.AsOf.In(loc).Format(asOfLayout)
	}
	var resp struct {
		JobID string `json:"job_id"`
	}
	if err := c.postJSON(ctx, "/v1/lab/kline/jobs", body, &resp); err != nil {
		return "", err
	}
	if strings.TrimSpace(resp.JobID) == "" {
		return "", fmt.Errorf("lab-api: empty job_id")
	}
	return resp.JobID, nil
}

func (c *HTTPClient) GetJob(ctx context.Context, jobID string) (Job, error) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return Job{}, fmt.Errorf("lab-api: empty job id")
	}
	var out Job
	if err := c.getJSON(ctx, "/v1/lab/jobs/"+jobID, &out); err != nil {
		return Job{}, err
	}
	if out.ID == "" {
		out.ID = jobID
	}
	return out, nil
}

// Health optionally probes GET /health (200 = reachable). Lab experiments do not require it.
func (c *HTTPClient) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/health", nil)
	if err != nil {
		return err
	}
	c.applyAuth(req)
	res, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode == http.StatusOK {
		return nil
	}
	return apiErrorFromResponse(res.StatusCode, b)
}

// FetchBars creates a kline job and polls GET /v1/lab/jobs/{id} until completed or failed.
// Retryable errors (5xx, poll timeout, transient job failure) trigger a fresh job with backoff.
func (c *HTTPClient) FetchBars(ctx context.Context, req KlineJobRequest) ([]any, error) {
	attempts := c.fetchAttempts()
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		bars, err := c.fetchBarsOnce(ctx, req)
		if err == nil {
			return bars, nil
		}
		lastErr = err
		if !IsRetryableFetchError(err) || attempt+1 >= attempts {
			return nil, err
		}
		if err := sleepFetchBackoff(ctx, attempt, c.retryBase()); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func (c *HTTPClient) fetchBarsOnce(ctx context.Context, req KlineJobRequest) ([]any, error) {
	jobID, err := c.CreateKlineJob(ctx, req)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(c.pollBudget())
	for {
		job, err := c.GetJob(ctx, jobID)
		if err != nil {
			return nil, err
		}
		switch job.Status {
		case JobCompleted:
			if job.Result == nil {
				return nil, fmt.Errorf("lab-api: job %s completed without result", jobID)
			}
			if job.Result.Bars == nil {
				return nil, fmt.Errorf("lab-api: job %s completed without result.bars", jobID)
			}
			return job.Result.Bars, nil
		case JobFailed:
			if job.Error != "" {
				return nil, fmt.Errorf("lab-api: job %s failed: %s", jobID, job.Error)
			}
			return nil, fmt.Errorf("lab-api: job %s failed", jobID)
		case JobPending, JobRunning:
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("lab-api: job %s poll timeout", jobID)
			}
			timer := time.NewTimer(c.pollInterval())
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
		default:
			return nil, fmt.Errorf("lab-api: job %s unknown status %q", jobID, job.Status)
		}
	}
}

func (c *HTTPClient) pollInterval() time.Duration {
	if c.PollEvery > 0 {
		return c.PollEvery
	}
	return 200 * time.Millisecond
}

func (c *HTTPClient) pollBudget() time.Duration {
	if c.PollBudget > 0 {
		return c.PollBudget
	}
	return 45 * time.Second
}

func (c *HTTPClient) postJSON(ctx context.Context, path string, body any, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyAuth(req)
	res, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return decodeJSON(res, out)
}

func (c *HTTPClient) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	c.applyAuth(req)
	res, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return decodeJSON(res, out)
}

func (c *HTTPClient) applyAuth(req *http.Request) {
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
}

func (c *HTTPClient) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func decodeJSON(res *http.Response, out any) error {
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return apiErrorFromResponse(res.StatusCode, b)
	}
	if out == nil {
		return nil
	}
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, out)
}

func apiErrorFromResponse(status int, body []byte) error {
	msg := parseErrorJSON(body)
	if msg == "" {
		msg = strings.TrimSpace(string(body))
	}
	if msg == "" {
		msg = http.StatusText(status)
	}
	return &APIError{StatusCode: status, Message: msg}
}

func parseErrorJSON(body []byte) string {
	var wrap struct {
		Error string `json:"error"`
	}
	if len(body) == 0 {
		return ""
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		return ""
	}
	return strings.TrimSpace(wrap.Error)
}
