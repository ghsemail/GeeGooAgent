package runtimeapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestDataProbeRequiresToken(t *testing.T) {
	handler := testCockpitHandler(t)
	body := bytes.NewBufferString(`{"checks":["quote"],"code":"600519.SH"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/data/probe", body)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDataProbeWithMockBot(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/getMarketNews":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":100,"data":{"market":"CN","text":"**1. headline**","items":[{"title":"headline"}],"sources_used":["eastmoney_sector"]}}`))
		case "/getStockNews":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":100,"data":{"code":"600519.SH","text":"**1. stock**","items":[{"title":"stock"}],"sources_used":["eastmoney_ann"]}}`))
		case "/getCurrentPrice":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":100,"price":1688.0}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mock.Close()

	handler := testCockpitHandlerWithConfig(t, &config.AppConfig{
		BaseURL:  mock.URL,
		APIKey:   "sk-test",
		UserMCPToken: "user-token",
	})
	body := bytes.NewBufferString(`{"checks":["market_news","stock_news","quote"],"market":"CN","code":"600519.SH","limit":3}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/data/probe", body)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["ok"] != true {
		t.Fatalf("ok=%v body=%s", resp["ok"], rec.Body.String())
	}
	results, ok := resp["results"].([]any)
	if !ok || len(results) != 3 {
		t.Fatalf("results=%v", resp["results"])
	}
}
