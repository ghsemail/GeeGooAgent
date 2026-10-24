package signalclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestIsRetryableFetchError_Classification(t *testing.T) {
	if IsRetryableFetchError(&APIError{StatusCode: 400, Message: "bad"}) {
		t.Fatal("400 not retryable")
	}
	if !IsRetryableFetchError(&APIError{StatusCode: 503, Message: "busy"}) {
		t.Fatal("503 retryable")
	}
	if IsRetryableFetchError(errors.New("lab-api: unsupported frequency")) {
		t.Fatal("unsupported not retryable")
	}
	if !IsRetryableFetchError(errors.New("lab-api: job j1 poll timeout")) {
		t.Fatal("poll timeout retryable")
	}
	if IsRetryableFetchError(context.Canceled) {
		t.Fatal("cancel not retryable")
	}
}

func TestHTTPClientFetchBarsRetriesOnPollTimeout(t *testing.T) {
	var creates atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/lab/kline/jobs":
			n := creates.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]string{"job_id": "j" + string(rune('0'+n))})
		case r.Method == http.MethodGet && stringsHasPrefixJob(r.URL.Path):
			id := r.URL.Path[len("/v1/lab/jobs/"):]
			if id == "j1" {
				_ = json.NewEncoder(w).Encode(Job{ID: id, Status: JobRunning})
				return
			}
			_ = json.NewEncoder(w).Encode(Job{
				ID: id, Status: JobCompleted,
				Result: &KlineJobResult{Bars: []any{map[string]any{"close": 1.0}}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "")
	c.PollEvery = 5 * time.Millisecond
	c.PollBudget = 15 * time.Millisecond
	c.FetchAttempts = 2
	c.RetryBase = 10 * time.Millisecond

	// First job never completes within poll budget → poll timeout → second job succeeds immediately.
	bars, err := c.FetchBars(context.Background(), KlineJobRequest{Code: "X", Frequency: "5m", MonthsBack: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(bars) != 1 {
		t.Fatalf("bars=%d", len(bars))
	}
	if creates.Load() < 2 {
		t.Fatalf("creates=%d want >=2", creates.Load())
	}
}

func TestHTTPClientFetchBarsDoesNotRetryBadRequest(t *testing.T) {
	var creates atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		creates.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid frequency"})
	}))
	defer srv.Close()
	c := NewHTTPClient(srv.URL, "")
	c.FetchAttempts = 3
	_, err := c.FetchBars(context.Background(), KlineJobRequest{Code: "X", Frequency: "bad", MonthsBack: 1})
	if err == nil {
		t.Fatal("expected error")
	}
	if creates.Load() != 1 {
		t.Fatalf("creates=%d want 1", creates.Load())
	}
}

func stringsHasPrefixJob(path string) bool {
	return len(path) > len("/v1/lab/jobs/")
}
