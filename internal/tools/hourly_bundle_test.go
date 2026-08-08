package tools_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestHourlyAnalysisBundleRunsParallel(t *testing.T) {
	t.Parallel()
	var inflight int32
	var peak int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := atomic.AddInt32(&inflight, 1)
		for {
			old := atomic.LoadInt32(&peak)
			if cur <= old || atomic.CompareAndSwapInt32(&peak, old, cur) {
				break
			}
		}
		time.Sleep(80 * time.Millisecond)
		atomic.AddInt32(&inflight, -1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 100, "message": "success",
			"data": map[string]any{
				"code": "00700.HK", "period": "hourly",
				"analysis_result": "bundle slot ok",
				"model": "test", "create_date": "2026-08-09",
			},
		})
	}))
	defer srv.Close()

	client := mcp.NewClient(srv.URL, "sk-test", mcp.Options{AllowedHosts: []string{"127.0.0.1"}})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{HTTP: tools.TestHTTPBackends(client), WorkspaceRoot: t.TempDir()})

	start := time.Now()
	res := r.Execute(tools.CallRequest{Name: "get_hourly_analysis_bundle", Arguments: map[string]any{
		"name": "腾讯控股", "code": "00700.HK",
	}}, tools.Context{SessionID: "bundle", MCPToken: "mcp-user", DryRun: false})
	elapsed := time.Since(start)

	if res.Status != tools.StatusOK {
		t.Fatalf("expected ok, got %s: %s", res.Status, res.Summary)
	}
	for _, key := range []string{"price_analysis", "signal_analysis", "kline_analysis"} {
		if v, _ := res.Data[key].(string); v == "" {
			t.Fatalf("missing %s in %+v", key, res.Data)
		}
	}
	if peak < 2 {
		t.Fatalf("expected parallel MCP calls, peak inflight=%d", peak)
	}
	if elapsed > 220*time.Millisecond {
		t.Fatalf("expected parallel bundle faster than serial, took %v", elapsed)
	}
}
