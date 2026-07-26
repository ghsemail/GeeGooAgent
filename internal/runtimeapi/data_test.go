package runtimeapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestDataOverviewWithMockNode(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/v1/market/capabilities":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"server_id":"test","regions":["CN"]}`))
		case "/v1/news/sources":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"server_id":"test",
				"regions":{"CN":{"market_sources":[{"id":"sina","enabled":true}],"stock_sources":[{"id":"ann","enabled":true}]}},
				"cache_market_ttl_sec":600,
				"cache_stock_ttl_sec":300
			}`))
		case "/v1/news/health":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"server_id":"test","sources":[{"id":"sina","ok":true}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mock.Close()

	handler := testCockpitHandlerWithConfig(t, &config.AppConfig{
		DataNodes: []config.DataNodeConfig{{
			ID: "test", Label: "Test Node", BaseURL: mock.URL, Regions: []string{"CN"},
		}},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/data/overview?force=true", nil)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	summary, ok := body["summary"].(map[string]any)
	if !ok {
		t.Fatalf("summary=%v", body["summary"])
	}
	if int(summary["nodes_total"].(float64)) != 1 {
		t.Fatalf("nodes_total=%v", summary["nodes_total"])
	}
	nodes, ok := body["nodes"].([]any)
	if !ok || len(nodes) != 1 {
		t.Fatalf("nodes=%v", body["nodes"])
	}
}

func TestDataNodeNewsSourcesProxy(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/news/sources" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"server_id":"test","regions":{}}`))
	}))
	defer mock.Close()

	handler := testCockpitHandlerWithConfig(t, &config.AppConfig{
		DataNodes: []config.DataNodeConfig{{ID: "cn", BaseURL: mock.URL}},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/data/nodes/cn/news/sources", nil)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["node_id"] != "cn" {
		t.Fatalf("node_id=%v", body["node_id"])
	}
}

func TestDataNodeNotFound(t *testing.T) {
	handler := testCockpitHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/data/nodes/missing/news/health", nil)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
