package signalclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
