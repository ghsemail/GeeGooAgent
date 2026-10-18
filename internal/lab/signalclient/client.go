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

// Client talks to GeeGooSignal lab-api (:3250). Task lifecycle is always queried via lab-api.
type Client interface {
	CreateKlineJob(ctx context.Context, req KlineJobRequest) (jobID string, err error)
	GetJob(ctx context.Context, jobID string) (Job, error)
}

// KlineJobRequest asks lab-api to fetch OHLCV (取数窗).
type KlineJobRequest struct {
	Code       string     `json:"code"`
	Frequency  string     `json:"frequency"`
	MonthsBack int        `json:"months_back"`
	AsOf       *time.Time `json:"as_of,omitempty"`
}

// Job is a lab-api async task snapshot.
type Job struct {
	ID     string         `json:"id"`
	Status JobStatus      `json:"status"`
	Error  string         `json:"error,omitempty"`
	Result *KlineJobResult `json:"result,omitempty"`
}

// JobStatus matches Signal lab-api task states (align with Signal implementation).
type JobStatus string

const (
	JobPending   JobStatus = "pending"
	JobRunning   JobStatus = "running"
	JobCompleted JobStatus = "completed"
	JobFailed    JobStatus = "failed"
)

// KlineJobResult is returned when status=completed.
type KlineJobResult struct {
	Bars []any `json:"bars"`
}

// HTTPClient implements Client against GeeGooSignal lab-api.
type HTTPClient struct {
	BaseURL    string
	APIKey     string
	HTTP       *http.Client
	PollEvery  time.Duration
	PollBudget time.Duration
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

func (c *HTTPClient) CreateKlineJob(ctx context.Context, req KlineJobRequest) (string, error) {
	body := map[string]any{
		"code":        strings.TrimSpace(req.Code),
		"frequency":   strings.TrimSpace(req.Frequency),
		"months_back": req.MonthsBack,
	}
	if req.AsOf != nil && !req.AsOf.IsZero() {
		loc, _ := time.LoadLocation("Asia/Shanghai")
		body["as_of"] = req.AsOf.In(loc).Format("2006-01-02 15:04:05")
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

// FetchBars creates a kline job and polls until completion (for Agent lab orchestration).
func (c *HTTPClient) FetchBars(ctx context.Context, req KlineJobRequest) ([]any, error) {
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
		msg := strings.TrimSpace(string(b))
		if msg == "" {
			msg = res.Status
		}
		return fmt.Errorf("lab-api: HTTP %d: %s", res.StatusCode, msg)
	}
	if out == nil {
		return nil
	}
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, out)
}
