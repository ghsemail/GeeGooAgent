package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newContractTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, "bearer", Options{
		AllowedHosts: []string{"127.0.0.1", "localhost"},
		MaxRetries:   1,
		Sleep:        func(time.Duration) {},
	})
}

func decodeClientCode(err error) (int, bool) {
	var ce *ClientError
	if errors.As(err, &ce) && ce.APICode != nil {
		return *ce.APICode, true
	}
	return 0, false
}

// N1: 000300.SH + legal date → 200, echo date, market=CN.
func TestCheckTradingDayOnDate_AppContract_N1_CN(t *testing.T) {
	client := newContractTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 100, "message": "success",
			"data": map[string]any{
				"is_trading_day": true,
				"date":           "2026-11-16",
				"market":         "CN",
				"code":           "000300.SH",
			},
		})
	})
	td, err := client.CheckTradingDayOnDate(context.Background(), "tok", "000300.SH", "2026-11-16")
	if err != nil || td == nil || !td.IsTradingDay {
		t.Fatalf("err=%v td=%+v", err, td)
	}
	q := CheckTradingDayQuery{Code: "000300.SH", Date: "2026-11-16"}
	if err := ValidateCheckTradingDayData(q, td); err != nil {
		t.Fatal(err)
	}
}

// N2: BTC-USDT.CRYPTO + date → market CRYPTO, is_trading_day true.
func TestCheckTradingDayOnDate_AppContract_N2_Crypto(t *testing.T) {
	client := newContractTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 100, "message": "success",
			"data": map[string]any{
				"is_trading_day": true,
				"date":           "2026-11-16",
				"market":         "CRYPTO",
				"code":           "BTC-USDT.CRYPTO",
			},
		})
	})
	td, err := client.CheckTradingDayOnDate(context.Background(), "tok", "BTC-USDT.CRYPTO", "2026-11-16")
	if err != nil || td == nil || !td.IsTradingDay || td.Market != "CRYPTO" {
		t.Fatalf("err=%v td=%+v", err, td)
	}
}

// N3: invalid date → HTTP 400, body.code=400, no success data.
func TestCheckTradingDayOnDate_AppContract_N3_InvalidDate(t *testing.T) {
	client := newContractTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    400,
			"message": "无效的 date，需为 YYYY-MM-DD",
		})
	})
	_, err := client.CheckTradingDayOnDate(context.Background(), "tok", "000300.SH", "2026/11/16")
	if err == nil {
		t.Fatal("expected error")
	}
	if api, ok := decodeClientCode(err); !ok || api != 400 {
		t.Fatalf("want body code 400, err=%v", err)
	}
}

// L1: omit date — four fields, no echo validation on consumer.
func TestCheckTradingDay_AppContract_L1_OmitDate(t *testing.T) {
	client := newContractTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 100, "message": "success",
			"data": map[string]any{
				"is_trading_day": true,
				"date":           "2026-11-17",
				"market":         "HK",
				"code":           "00700.HK",
			},
		})
	})
	td, err := client.CheckTradingDay(context.Background(), "tok", "00700.HK")
	if err != nil || td == nil || td.Market != "HK" {
		t.Fatalf("err=%v td=%+v", err, td)
	}
}

// O1a: legacy echo / wrong market on 200 → consumer validation error (daily_macro path).
func TestValidateCheckTradingDayData_AppContract_O1(t *testing.T) {
	q := CheckTradingDayQuery{Code: "000300.SH", Date: "2026-11-16"}
	if err := ValidateCheckTradingDayData(q, &TradingDayData{
		IsTradingDay: true, Date: "2026-11-17", Market: "CN", Code: "000300.SH",
	}); err == nil {
		t.Fatal("today echo mismatch")
	}
	q = CheckTradingDayQuery{Code: "BTC-USDT.CRYPTO", Date: "2026-11-16"}
	if err := ValidateCheckTradingDayData(q, &TradingDayData{
		IsTradingDay: true, Date: "2026-11-16", Market: "US", Code: "BTC-USDT.CRYPTO",
	}); err == nil {
		t.Fatal("crypto must not echo US")
	}
}

func TestCheckTradingDayOnDate_AppContract_O1_HTTP400UnknownMarket(t *testing.T) {
	client := newContractTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 400, "message": "无法从 code 推断 market",
		})
	})
	_, err := client.CheckTradingDayOnDate(context.Background(), "tok", "FOO.BAR", "2026-11-16")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCheckTradingDayOnDate_AppContract_HTTP500ReturnsError(t *testing.T) {
	client := newContractTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":500,"message":"calendar backend failed"}`))
	})
	_, err := client.CheckTradingDayOnDate(context.Background(), "tok", "000300.SH", "2026-11-16")
	if err == nil {
		t.Fatal("expected transport/API error, not success data")
	}
}