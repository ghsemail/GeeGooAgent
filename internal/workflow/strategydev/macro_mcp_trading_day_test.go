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

func TestFailClosedMacroTradingDayChecker_Errors(t *testing.T) {
	chk := FailClosedMacroTradingDayChecker("no mcp")
	_, err := chk.IsExchangeTradingDay(context.Background(), "000300.SH", "2026-11-16")
	if err == nil {
		t.Fatal("expected error")
	}
}