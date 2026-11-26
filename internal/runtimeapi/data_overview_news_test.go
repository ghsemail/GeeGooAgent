package runtimeapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestDataOverviewFleetDedupesSharedSources(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/v1/market/capabilities":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"regions":["CRYPTO"]}`))
		case "/v1/news/sources":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"catalog":[
					{"id":"websearch","enabled":true,"shared":true},
					{"id":"x_public","enabled":true,"shared":true,"probe_exempt":true},
					{"id":"binance_headlines","enabled":true}
				]
			}`))
		case "/v1/news/health":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"sources":[
				{"id":"websearch","ok":true},
				{"id":"x_public","ok":true},
				{"id":"binance_headlines","ok":false,"error":"stub"}
			]}`))
		case "/v1/crypto/health":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true,"source":"binance"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mock.Close()

	payload := `{
		"catalog":[
			{"id":"websearch","enabled":true,"shared":true},
			{"id":"x_public","enabled":true,"shared":true,"probe_exempt":true},
			{"id":"eastmoney_sector","enabled":true}
		]
	}`
	healthPayload := `{"sources":[
		{"id":"websearch","ok":true},
		{"id":"x_public","ok":true},
		{"id":"eastmoney_sector","ok":true}
	]}`

	mockCN := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/v1/market/capabilities":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"regions":["CN"]}`))
		case "/v1/news/sources":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(payload))
		case "/v1/news/health":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(healthPayload))
		case "/v1/futu/health":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockCN.Close()

	handler := testCockpitHandlerWithConfig(t, &config.AppConfig{
		DataNodes: []config.DataNodeConfig{
			{ID: "ashare-cn", Label: "A股", BaseURL: mockCN.URL, Regions: []string{"CN"}},
			{ID: "crypto", Label: "Crypto", BaseURL: mock.URL, Regions: []string{"CRYPTO"}},
		},
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
	summary := body["summary"].(map[string]any)
	if int(summary["sources_total"].(float64)) != 3 {
		t.Fatalf("sources_total=%v want 3 (websearch,eastmoney_sector,binance_headlines; x_public exempt)", summary["sources_total"])
	}
	if int(summary["sources_healthy"].(float64)) != 2 {
		t.Fatalf("sources_healthy=%v want 2", summary["sources_healthy"])
	}
}
