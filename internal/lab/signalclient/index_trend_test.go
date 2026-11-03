package signalclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchIndexTrendMock(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/lab/index-trend" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("auth=%q", r.Header.Get("Authorization"))
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "00700.HK", "frequency": got["frequency"], "strategy_version": "alpha",
			"months_back": got["months_back"], "bar_count": 42, "last_bar_time": "2026-11-01 15:00:00",
			"trends": []map[string]any{
				{"name": "ADX趋势", "index": "ADX", "type": "flag", "stance": "long", "value": 1},
			},
			"summary": map[string]any{"long": 2, "short": 1, "flat": 0, "net": 1},
		})
	}))
	defer srv.Close()

	client := NewHTTPClient(srv.URL, "test-key")
	asOf := time.Date(2026, 11, 1, 15, 0, 0, 0, time.UTC)
	resp, err := client.FetchIndexTrend(context.Background(), IndexTrendRequest{
		Code: "00700.HK", Frequency: "60m", MonthsBack: 3, StrategyVersion: "α", AsOf: &asOf,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.BarCount != 42 || resp.Summary.Net != 1 {
		t.Fatalf("resp=%+v", resp)
	}
	if got["strategy_version"] != "alpha" {
		t.Fatalf("strategy_version=%v", got["strategy_version"])
	}
	if got["as_of"] == nil {
		t.Fatal("expected as_of in request")
	}
}

func TestFetchIndexTrendValidation(t *testing.T) {
	client := NewHTTPClient("http://unused", "k")
	_, err := client.FetchIndexTrend(context.Background(), IndexTrendRequest{Code: "X"})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if api, ok := err.(*APIError); !ok || api.StatusCode != 400 {
		t.Fatalf("err=%v", err)
	}
}
