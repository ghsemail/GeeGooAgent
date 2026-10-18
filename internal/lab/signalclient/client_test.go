package signalclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPClientFetchBarsPollsJob(t *testing.T) {
	var polls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/lab/kline/jobs":
			_ = json.NewEncoder(w).Encode(map[string]string{"job_id": "j1"})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/lab/jobs/j1":
			n := polls.Add(1)
			if n < 2 {
				_ = json.NewEncoder(w).Encode(Job{ID: "j1", Status: JobRunning})
				return
			}
			_ = json.NewEncoder(w).Encode(Job{
				ID: "j1", Status: JobCompleted,
				Result: &KlineJobResult{Bars: []any{map[string]any{"close": 1.0}}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "test-key")
	c.PollEvery = 10
	c.PollBudget = 2 * time.Second
	bars, err := c.FetchBars(context.Background(), KlineJobRequest{Code: "AAPL", Frequency: "60m", MonthsBack: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(bars) != 1 {
		t.Fatalf("bars=%d", len(bars))
	}
}

func TestHTTPClientHealthOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	c := NewHTTPClient(srv.URL, "")
	if err := c.Health(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPClientCreateJobAPIErrorJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid frequency"})
	}))
	defer srv.Close()
	c := NewHTTPClient(srv.URL, "")
	_, err := c.CreateKlineJob(context.Background(), KlineJobRequest{Code: "X", Frequency: "bad", MonthsBack: 1})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errorsAsAPIError(err, &apiErr) {
		t.Fatalf("want *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 400 || apiErr.Message != "invalid frequency" {
		t.Fatalf("apiErr=%+v", apiErr)
	}
}

func TestHTTPClientFetchBarsFailedJobReturns200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/lab/kline/jobs":
			_ = json.NewEncoder(w).Encode(map[string]string{"job_id": "j2"})
		case "/v1/lab/jobs/j2":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(Job{ID: "j2", Status: JobFailed, Error: "worker oom"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := NewHTTPClient(srv.URL, "")
	_, err := c.FetchBars(context.Background(), KlineJobRequest{Code: "X", Frequency: "5m", MonthsBack: 1})
	if err == nil || !strings.Contains(err.Error(), "worker oom") {
		t.Fatalf("err=%v", err)
	}
}

func errorsAsAPIError(err error, target **APIError) bool {
	if err == nil {
		return false
	}
	if ae, ok := err.(*APIError); ok {
		*target = ae
		return true
	}
	return false
}
