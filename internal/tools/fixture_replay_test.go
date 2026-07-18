package tools_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// Fixture: a realistic getAllSmartTrades envelope with one SpaceX SmartTrade.
const listSmartTradesFixture = `{
  "code": 100,
  "message": "success",
  "data": [
    {
      "bot_id": "st-spacex-1",
      "botname": "SpaceX DCA",
      "stock_name": "SPACEX",
      "code": "SPACEX.US",
      "bot_type": "SmartTrade",
      "status": "running"
    }
  ]
}`

func TestFixtureListSmartTradesReplay(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getAllSmartTrades" {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["mcp_token"] != "mcp-user-fixtures" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(listSmartTradesFixture))
	}))
	defer srv.Close()

	client := mcp.NewClient(srv.URL, "sk-test", mcp.Options{AllowedHosts: []string{"127.0.0.1"}})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{HTTP: tools.TestHTTPBackends(client), WorkspaceRoot: t.TempDir()})

	res := r.Execute(tools.CallRequest{Name: "list_smart_trades", Arguments: map[string]any{}}, tools.Context{
		SessionID: "s1", MCPToken: "mcp-user-fixtures", DryRun: false,
	})
	if res.Status != tools.StatusOK {
		t.Fatalf("status=%s summary=%s", res.Status, res.Summary)
	}
	items, _ := res.Data["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	first := items[0].(map[string]any)
	if first["bot_id"] != "st-spacex-1" {
		t.Fatalf("bot_id = %v", first["bot_id"])
	}
	if first["stock_name"] != "SPACEX" {
		t.Fatalf("stock_name = %v", first["stock_name"])
	}
	if res.Meta == nil || res.Meta["api_code"] != float64(100) {
		t.Fatalf("meta api_code missing: %+v", res.Meta)
	}
}

// Fixture: empty list but code=100 → should classify as Skip with data gap.
const emptySmartTradesFixture = `{
  "code": 100,
  "message": "success",
  "data": []
}`

func TestFixtureListSmartTradesEmptyClassifiedAsSkip(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(emptySmartTradesFixture))
	}))
	defer srv.Close()

	client := mcp.NewClient(srv.URL, "sk-test", mcp.Options{AllowedHosts: []string{"127.0.0.1"}})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{HTTP: tools.TestHTTPBackends(client), WorkspaceRoot: t.TempDir()})

	res := r.Execute(tools.CallRequest{Name: "list_smart_trades", Arguments: map[string]any{}}, tools.Context{
		SessionID: "s2", MCPToken: "mcp-user", DryRun: false,
	})
	if res.Status != tools.StatusSkip {
		t.Fatalf("expected skip for empty list, got %s: %s", res.Status, res.Summary)
	}
}

// Fixture: empty analysis_result but code=100 → Skip with data gap.
const emptyAnalysisFixture = `{
  "code": 100,
  "message": "success",
  "data": {
    "code": "00700.HK",
    "period": "weekly",
    "analysis_result": "",
    "model": "deepseek-v4-pro",
    "create_date": "2026-07-04"
  }
}`

func TestFixtureGetMCPAnalysisEmptyResultClassifiedAsSkip(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(emptyAnalysisFixture))
	}))
	defer srv.Close()

	client := mcp.NewClient(srv.URL, "sk-test", mcp.Options{AllowedHosts: []string{"127.0.0.1"}})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{HTTP: tools.TestHTTPBackends(client), WorkspaceRoot: t.TempDir()})

	res := r.Execute(tools.CallRequest{Name: "get_mcp_analysis", Arguments: map[string]any{
		"name": "腾讯控股", "code": "00700.HK", "period": "weekly",
	}}, tools.Context{SessionID: "s3", MCPToken: "mcp-user", DryRun: false})
	if res.Status != tools.StatusSkip {
		t.Fatalf("expected skip for empty analysis, got %s: %s", res.Status, res.Summary)
	}
}

const emptyGridStrategyFixture = `{
  "code": 100,
  "message": "GRID策略生成完成",
  "data": {
    "suitable": false,
    "reason": "暂不适合网格",
    "model_name": "test-model"
  }
}`

const validGridStrategyFixture = `{
  "code": 100,
  "message": "GRID策略生成完成",
  "data": {
    "suitable": true,
    "param": {"grid_num": 7, "lower_limit_price": 500, "upper_limit_price": 640},
    "reason": "适合网格"
  }
}`

func TestFixtureGenerateGridStrategyEmptyClassifiedAsSkip(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/generateGridStrategy" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(emptyGridStrategyFixture))
	}))
	defer srv.Close()

	client := mcp.NewClient(srv.URL, "sk-test", mcp.Options{AllowedHosts: []string{"127.0.0.1"}})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{HTTP: tools.TestHTTPBackends(client), WorkspaceRoot: t.TempDir()})

	res := r.Execute(tools.CallRequest{Name: "generate_grid_strategy", Arguments: map[string]any{
		"code": "00700.HK", "name": "腾讯控股",
	}}, tools.Context{SessionID: "s-grid-empty", MCPToken: "mcp-user", DryRun: false})
	if res.Status != tools.StatusSkip {
		t.Fatalf("expected skip for empty grid strategy, got %s: %s", res.Status, res.Summary)
	}
}

func TestFixtureGenerateGridStrategyWithParamOK(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/generateGridStrategy" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(validGridStrategyFixture))
	}))
	defer srv.Close()

	client := mcp.NewClient(srv.URL, "sk-test", mcp.Options{AllowedHosts: []string{"127.0.0.1"}})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{HTTP: tools.TestHTTPBackends(client), WorkspaceRoot: t.TempDir()})

	res := r.Execute(tools.CallRequest{Name: "generate_grid_strategy", Arguments: map[string]any{
		"code": "00700.HK", "name": "腾讯控股",
	}}, tools.Context{SessionID: "s-grid-ok", MCPToken: "mcp-user", DryRun: false})
	if res.Status != tools.StatusOK {
		t.Fatalf("expected ok, got %s: %s", res.Status, res.Summary)
	}
	param, _ := res.Data["param"].(map[string]any)
	if param == nil || param["grid_num"] == nil {
		t.Fatalf("expected param in data: %+v", res.Data)
	}
}

const emptyDCAStrategyFixture = `{
  "code": 100,
  "message": "DCA策略生成完成",
  "data": {
    "signal_id": "662d0424c4cee7ffb800d0af",
    "signal": {"buy_signal": [], "name": "test"},
    "trend_conclusion": {"suitable": false, "reason": "不适合"}
  }
}`

const validDCAStrategyFixture = `{
  "code": 100,
  "message": "DCA策略生成完成",
  "data": {
    "signal_id": "662d0424c4cee7ffb800d0af",
    "signal": {
      "buy_signal": [{"index": "SAR", "type": "signal", "param": {"acceleration": "0.02"}}],
      "name": "SAR信号"
    }
  }
}`

func TestFixtureGenerateDCAStrategyEmptyClassifiedAsSkip(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/generateDCAStrategy" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(emptyDCAStrategyFixture))
	}))
	defer srv.Close()

	client := mcp.NewClient(srv.URL, "sk-test", mcp.Options{AllowedHosts: []string{"127.0.0.1"}})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{HTTP: tools.TestHTTPBackends(client), WorkspaceRoot: t.TempDir()})

	res := r.Execute(tools.CallRequest{Name: "generate_dca_strategy", Arguments: map[string]any{
		"code": "00700.HK", "name": "腾讯控股", "signal_id": "662d0424c4cee7ffb800d0af",
	}}, tools.Context{SessionID: "s-dca-empty", MCPToken: "mcp-user", DryRun: false})
	if res.Status != tools.StatusSkip {
		t.Fatalf("expected skip for empty dca strategy, got %s: %s", res.Status, res.Summary)
	}
}

func TestFixtureGenerateDCAStrategyWithBuySignalOK(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/generateDCAStrategy" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(validDCAStrategyFixture))
	}))
	defer srv.Close()

	client := mcp.NewClient(srv.URL, "sk-test", mcp.Options{AllowedHosts: []string{"127.0.0.1"}})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{HTTP: tools.TestHTTPBackends(client), WorkspaceRoot: t.TempDir()})

	res := r.Execute(tools.CallRequest{Name: "generate_dca_strategy", Arguments: map[string]any{
		"code": "00700.HK", "name": "腾讯控股", "signal_id": "662d0424c4cee7ffb800d0af",
	}}, tools.Context{SessionID: "s-dca-ok", MCPToken: "mcp-user", DryRun: false})
	if res.Status != tools.StatusOK {
		t.Fatalf("expected ok, got %s: %s", res.Status, res.Summary)
	}
}
