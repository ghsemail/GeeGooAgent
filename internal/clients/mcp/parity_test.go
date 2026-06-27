package mcp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
)

const testAPIKey = "sk-test-key"
const testMCPToken = "mcp-token"

func fixturePath(name string) string {
	// tests/fixtures/geegoo relative to module root
	return filepath.Join("..", "..", "..", "tests", "fixtures", "geegoo", name)
}

func loadFixture(t *testing.T, name string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(fixturePath(name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	return data
}

func newTestClient(t *testing.T, handler http.HandlerFunc) *mcp.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return mcp.NewClient(srv.URL, testAPIKey, mcp.Options{
		AllowedHosts: []string{"127.0.0.1", "localhost"},
		MaxRetries:   3,
		RetryWait:    0,
		Sleep:        func(time.Duration) {},
	})
}

func envelopeHandler(path string, fixture string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.NotFound(w, r)
			return
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer "+testAPIKey {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		raw, _ := os.ReadFile(fixturePath(fixture))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}
}

// parityCases: 10 high-frequency MCP tools aligned with Python integration tests.
func TestParityHighFrequencyTools(t *testing.T) {
	ctx := context.Background()

	t.Run("checkTradingDay", func(t *testing.T) {
		client := newTestClient(t, envelopeHandler("/checkTradingDay", "check_trading_day_ok.json"))
		result, err := client.CheckTradingDay(ctx, testMCPToken, "00700.HK")
		if err != nil {
			t.Fatal(err)
		}
		if !result.IsTradingDay || result.Market != "HK" {
			t.Fatalf("unexpected: %+v", result)
		}
	})

	t.Run("getReportBotCodes", func(t *testing.T) {
		client := newTestClient(t, envelopeHandler("/getReportBotCodes", "get_report_bot_codes_ok.json"))
		bots, err := client.GetReportBotCodes(ctx, testMCPToken)
		if err != nil {
			t.Fatal(err)
		}
		if len(bots) != 1 || bots[0].StockName != "腾讯控股" || bots[0].BotType != "DCA" || bots[0].BotID == "" {
			t.Fatalf("unexpected: %+v", bots)
		}
	})

	t.Run("getCapitalFlow", func(t *testing.T) {
		client := newTestClient(t, envelopeHandler("/getCapitalFlow", "get_capital_flow_ok.json"))
		flows, err := client.GetCapitalFlow(ctx, testMCPToken, "00700.HK", "DAY", "")
		if err != nil {
			t.Fatal(err)
		}
		if len(flows) != 1 || flows[0].MainInFlow != -106682800.0 {
			t.Fatalf("unexpected: %+v", flows)
		}
	})

	t.Run("getCapitalDistribution", func(t *testing.T) {
		client := newTestClient(t, envelopeHandler("/getCapitalDistribution", "get_capital_distribution_ok.json"))
		dist, err := client.GetCapitalDistribution(ctx, testMCPToken, "00700.HK")
		if err != nil {
			t.Fatal(err)
		}
		if dist.CapitalInSuper != 1000000.0 || dist.UpdateTime != "2026-04-27 15:59:59" {
			t.Fatalf("unexpected: %+v", dist)
		}
	})

	t.Run("getBotYesterdayAttitude", func(t *testing.T) {
		client := newTestClient(t, envelopeHandler("/getBotYesterdayAttitude", "get_bot_yesterday_attitude_ok.json"))
		att, err := client.GetBotYesterdayAttitude(ctx, testMCPToken, "662f3e12ab45cd7890ef1234", "cn")
		if err != nil {
			t.Fatal(err)
		}
		if att.Attitude != "bullish" || !att.Found {
			t.Fatalf("unexpected: %+v", att)
		}
	})

	t.Run("getMCPAnalysis", func(t *testing.T) {
		client := newTestClient(t, envelopeHandler("/getMCPAnalysis", "get_mcp_analysis_ok.json"))
		result, err := client.GetMCPAnalysis(ctx, testMCPToken, "恒生指数", "800000.HK", "69ec7035b9ccd3d9befc6c23", "hourly", "cn")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(result.AnalysisResult, "周线分析") {
			t.Fatalf("unexpected analysis: %q", result.AnalysisResult)
		}
	})

	t.Run("createPreMarketReport", func(t *testing.T) {
		client := newTestClient(t, envelopeHandler("/createPreMarketReport", "create_pre_market_report_ok.json"))
		result, err := client.CreatePreMarketReport(ctx, testMCPToken, map[string]any{
			"code": "00700.HK", "stock_name": "腾讯控股", "bot_id": "bot-1",
			"bot_name": "DCA", "bot_type": "DCA", "result": "long",
			"confidence": "high", "reason": "test", "suggestion": "buy", "report": "report body",
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.ReportID != "680bc8e7f54cf8a14f82a8a2" {
			t.Fatalf("unexpected report_id: %q", result.ReportID)
		}
	})

	t.Run("getStockDailyReports", func(t *testing.T) {
		client := newTestClient(t, envelopeHandler("/getStockDailyReports", "get_stock_daily_reports_ok.json"))
		reports, err := client.GetStockDailyReports(ctx, testMCPToken, "00700.HK", "2026-06-05")
		if err != nil {
			t.Fatal(err)
		}
		if len(reports.PreMarket) != 1 {
			t.Fatalf("unexpected: %+v", reports)
		}
		if id, _ := reports.PreMarket[0]["report_id"].(string); id == "" {
			t.Fatalf("missing report_id: %+v", reports.PreMarket[0])
		}
	})

	t.Run("searchCode", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/searchCode" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"code":"00700.HK","name":{"en":"TENCENT","init":"腾讯控股","zh_hk":"騰訊控股"},"market":"HK","lot_size":100,"stock_type":"STOCK"}]`))
		}
		client := newTestClient(t, handler)
		items, err := client.SearchCode(ctx, "00700", []string{"HK"})
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Code != "00700.HK" || items[0].Name != "腾讯控股" {
			t.Fatalf("unexpected: %+v", items)
		}
	})

	t.Run("getCurrentPrice", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/getCurrentPrice" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"price":99.5}`))
		}
		client := newTestClient(t, handler)
		price, err := client.GetCurrentPrice(ctx, testMCPToken, "00700.HK")
		if err != nil {
			t.Fatal(err)
		}
		if price.Price != 99.5 {
			t.Fatalf("unexpected price: %v", price.Price)
		}
	})
}

func TestAPIErrorParity(t *testing.T) {
	ctx := context.Background()
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":102,"message":"invalid token"}`))
	}
	client := newTestClient(t, handler)
	_, err := client.CheckTradingDay(ctx, "bad-token", "00700.HK")
	if err == nil {
		t.Fatal("expected error")
	}
	ce, ok := err.(*mcp.ClientError)
	if !ok || ce.APICode == nil || *ce.APICode != 102 {
		t.Fatalf("expected api code 102, got %v", err)
	}
}

func TestBotYesterdayAttitude404Neutral(t *testing.T) {
	ctx := context.Background()
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":105,"message":"未找到昨天的 attitude 记录"}`))
	}
	client := newTestClient(t, handler)
	att, err := client.GetBotYesterdayAttitude(ctx, testMCPToken, "bot-missing", "cn")
	if err != nil {
		t.Fatal(err)
	}
	if att.Attitude != "neutral" || att.Found {
		t.Fatalf("unexpected: %+v", att)
	}
}

func TestDisallowedHost(t *testing.T) {
	client := mcp.NewClient("http://evil.example.com:3120", testAPIKey, mcp.Options{
		AllowedHosts: []string{"118.195.135.97"},
	})
	_, err := client.CheckTradingDay(context.Background(), testMCPToken, "00700.HK")
	if err == nil {
		t.Fatal("expected sandbox error")
	}
	if _, ok := err.(*mcp.SandboxError); !ok {
		t.Fatalf("expected SandboxError, got %T: %v", err, err)
	}
}

func TestBearerHeaderOnRequest(t *testing.T) {
	var gotAuth string
	var gotBody map[string]any
	handler := func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		raw, _ := os.ReadFile(fixturePath("check_trading_day_ok.json"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}
	client := newTestClient(t, handler)
	_, err := client.CheckTradingDay(context.Background(), testMCPToken, "00700.HK")
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer "+testAPIKey {
		t.Fatalf("auth header = %q", gotAuth)
	}
	if gotBody["mcp_token"] != testMCPToken {
		t.Fatalf("body mcp_token = %v", gotBody["mcp_token"])
	}
}

func TestFixturesLoadable(t *testing.T) {
	names := []string{
		"check_trading_day_ok.json",
		"get_report_bot_codes_ok.json",
		"get_capital_flow_ok.json",
		"get_capital_distribution_ok.json",
		"get_bot_yesterday_attitude_ok.json",
		"get_mcp_analysis_ok.json",
		"create_pre_market_report_ok.json",
		"get_stock_daily_reports_ok.json",
	}
	for _, name := range names {
		data := loadFixture(t, name)
		if code, ok := data["code"].(float64); !ok || int(code) != 100 {
			t.Fatalf("%s: expected code 100, got %v", name, data["code"])
		}
	}
}
