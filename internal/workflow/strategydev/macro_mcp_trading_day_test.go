package strategydev

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
)

func TestMCPMacroTradingDayChecker_IsExchangeTradingDay(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 100, "message": "success",
			"data": map[string]any{
				"is_trading_day": true, "date": "2026-11-16", "market": "CN", "code": "000300.SH",
			},
		})
	}))
	defer srv.Close()
	client := mcp.NewClient(srv.URL, "key", mcp.Options{
		AllowedHosts: []string{"127.0.0.1", "localhost"},
		MaxRetries:   1,
		Sleep:        func(time.Duration) {},
	})
	chk := NewMCPMacroTradingDayChecker(client, "user-tok")
	ok, err := chk.IsExchangeTradingDay(context.Background(), "000300.SH", "2026-11-16")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestMCPMacroTradingDayChecker_LegacyTodayEchoRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 100, "message": "success",
			"data": map[string]any{
				"is_trading_day": true,
				"date":           "2026-11-17",
				"market":         "CN",
				"code":           "000300.SH",
			},
		})
	}))
	defer srv.Close()
	client := mcp.NewClient(srv.URL, "key", mcp.Options{
		AllowedHosts: []string{"127.0.0.1", "localhost"},
		MaxRetries:   1,
		Sleep:        func(time.Duration) {},
	})
	chk := NewMCPMacroTradingDayChecker(client, "user-tok")
	_, err := chk.IsExchangeTradingDay(context.Background(), "000300.SH", "2020-01-02")
	if err == nil {
		t.Fatal("expected fail-closed when response date is today-only echo")
	}
}

func TestMCPMacroTradingDayChecker_AppContract_O1_Integration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 100, "message": "success",
			"data": map[string]any{
				"is_trading_day": true, "date": "2026-11-16", "market": "HK", "code": "BTC-USDT.CRYPTO",
			},
		})
	}))
	defer srv.Close()
	client := mcp.NewClient(srv.URL, "key", mcp.Options{
		AllowedHosts: []string{"127.0.0.1", "localhost"},
		MaxRetries:   1,
		Sleep:        func(time.Duration) {},
	})
	chk := NewMCPMacroTradingDayChecker(client, "user-tok")
	_, err := chk.IsExchangeTradingDay(context.Background(), "BTC-USDT.CRYPTO", "2026-11-16")
	if err == nil {
		t.Fatal("expected fail-closed on CRYPTO with HK market echo")
	}
}

func TestMCPMacroTradingDayChecker_HTTP500NotTradingDayFalse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":500,"message":"fail"}`))
	}))
	defer srv.Close()
	client := mcp.NewClient(srv.URL, "key", mcp.Options{
		AllowedHosts: []string{"127.0.0.1", "localhost"},
		MaxRetries:   1,
		Sleep:        func(time.Duration) {},
	})
	chk := NewMCPMacroTradingDayChecker(client, "user-tok")
	ok, err := chk.IsExchangeTradingDay(context.Background(), "000300.SH", "2026-11-16")
	if err == nil || ok {
		t.Fatalf("want error not ok=%v", ok)
	}
}

func TestFailClosedMacroTradingDayChecker_Errors(t *testing.T) {
	chk := FailClosedMacroTradingDayChecker("no mcp")
	_, err := chk.IsExchangeTradingDay(context.Background(), "000300.SH", "2026-11-16")
	if err == nil {
		t.Fatal("expected error")
	}
}