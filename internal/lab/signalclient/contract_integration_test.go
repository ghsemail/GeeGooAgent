package signalclient_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/lab/signalclient"
	"github.com/ghsemail/GeeGooAgent/internal/lab/signalclient/reference"
)

// Contract integration against reference lab-api (Signal PR#2 shape). Not the production Signal binary.
func TestContractAgainstReferenceLabAPI(t *testing.T) {
	srv := reference.NewServer()
	ts := httptest.NewServer(srv.HTTP.Handler)
	defer ts.Close()

	client := signalclient.NewHTTPClient(ts.URL, reference.DefaultAPIKey)
	client.PollEvery = 5 * time.Millisecond
	client.PollBudget = 10 * time.Second

	t.Run("health", func(t *testing.T) {
		if err := client.Health(context.Background()); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("auth_missing", func(t *testing.T) {
		bad := signalclient.NewHTTPClient(ts.URL, "")
		_, err := bad.CreateKlineJob(context.Background(), signalclient.KlineJobRequest{
			Code: "00700.HK", Frequency: "5m", MonthsBack: 1,
		})
		if err == nil {
			t.Fatal("expected auth error")
		}
		var apiErr *signalclient.APIError
		if !asAPIError(err, &apiErr) || apiErr.StatusCode != 401 {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("create_invalid_frequency", func(t *testing.T) {
		_, err := client.CreateKlineJob(context.Background(), signalclient.KlineJobRequest{
			Code: "00700.HK", Frequency: "bad", MonthsBack: 3,
		})
		if err == nil {
			t.Fatal("expected 400")
		}
		var apiErr *signalclient.APIError
		if !asAPIError(err, &apiErr) || apiErr.StatusCode != 400 {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("poll_completed_bars_shape", func(t *testing.T) {
		bars, err := client.FetchBars(context.Background(), signalclient.KlineJobRequest{
			Code: "00700.HK", Frequency: "60m", MonthsBack: 3,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(bars) == 0 {
			t.Fatal("expected bars")
		}
		row, ok := bars[0].(map[string]any)
		if !ok {
			t.Fatalf("bar type %T", bars[0])
		}
		for _, k := range []string{"time", "open", "high", "low", "close", "volume"} {
			if row[k] == nil {
				t.Fatalf("missing %s", k)
			}
		}
	})

	t.Run("poll_failed_job_http200", func(t *testing.T) {
		_, err := client.FetchBars(context.Background(), signalclient.KlineJobRequest{
			Code: "FAIL.HK", Frequency: "5m", MonthsBack: 1,
		})
		if err == nil || !strings.Contains(err.Error(), "simulated worker failure") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("get_job_not_found", func(t *testing.T) {
		_, err := client.GetJob(context.Background(), "missing-id")
		if err == nil {
			t.Fatal("expected 404")
		}
		var apiErr *signalclient.APIError
		if !asAPIError(err, &apiErr) || apiErr.StatusCode != 404 {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("as_of_truncation", func(t *testing.T) {
		loc, _ := time.LoadLocation("Asia/Shanghai")
		asOf := time.Date(2026, 6, 1, 15, 0, 0, 0, loc)
		bars, err := client.FetchBars(context.Background(), signalclient.KlineJobRequest{
			Code: "AAPL.US", Frequency: "60m", MonthsBack: 3, AsOf: &asOf,
		})
		if err != nil {
			t.Fatal(err)
		}
		for _, raw := range bars {
			row := raw.(map[string]any)
			ts, _ := time.ParseInLocation("2006-01-02 15:04:05", row["time"].(string), loc)
			if ts.After(asOf) {
				t.Fatalf("bar %v after as_of", row["time"])
			}
		}
	})

	t.Run("status_enum_path", func(t *testing.T) {
		jobID, err := client.CreateKlineJob(context.Background(), signalclient.KlineJobRequest{
			Code: "00700.HK", Frequency: "daily", MonthsBack: 2,
		})
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			job, err := client.GetJob(context.Background(), jobID)
			if err != nil {
				t.Fatal(err)
			}
			seen[string(job.Status)] = true
			if job.Status == signalclient.JobCompleted {
				if job.Result == nil || job.Result.Bars == nil {
					t.Fatal("completed must include result.bars")
				}
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		for _, st := range []string{"pending", "running", "completed"} {
			if !seen[st] {
				t.Fatalf("did not observe status %s (seen=%v)", st, seen)
			}
		}
	})
}

func asAPIError(err error, target **signalclient.APIError) bool {
	if err == nil {
		return false
	}
	if ae, ok := err.(*signalclient.APIError); ok {
		*target = ae
		return true
	}
	return false
}
