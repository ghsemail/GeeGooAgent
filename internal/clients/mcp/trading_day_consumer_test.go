package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func loadTradingDayFixture(t *testing.T, name string) *TradingDayData {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "trading_day", name))
	if err != nil {
		t.Fatal(err)
	}
	var td TradingDayData
	if err := json.Unmarshal(b, &td); err != nil {
		t.Fatal(err)
	}
	return &td
}

func TestValidateCheckTradingDayData_AppContractEchoOK(t *testing.T) {
	td := loadTradingDayFixture(t, "per_day_echo_ok.json")
	q := CheckTradingDayQuery{Code: "000300.SH", Date: "2024-01-02"}
	if err := ValidateCheckTradingDayData(q, td); err != nil {
		t.Fatal(err)
	}
}

func TestValidateCheckTradingDayData_LegacyTodayMismatch(t *testing.T) {
	td := loadTradingDayFixture(t, "legacy_today_mismatch.json")
	q := CheckTradingDayQuery{Code: "000300.SH", Date: "2020-07-01"}
	if err := ValidateCheckTradingDayData(q, td); err == nil {
		t.Fatal("expected fail-closed on date echo mismatch")
	}
}

func TestValidateCheckTradingDayData_MissingDateEcho(t *testing.T) {
	td := loadTradingDayFixture(t, "missing_date_echo.json")
	q := CheckTradingDayQuery{Code: "SPY.US", Date: "2024-11-15"}
	if err := ValidateCheckTradingDayData(q, td); err == nil {
		t.Fatal("expected missing date echo error")
	}
}

func TestValidateCheckTradingDayData_CryptoMustBeCRYPTO(t *testing.T) {
	td := &TradingDayData{
		IsTradingDay: true,
		Date:         "2024-06-01",
		Market:       "US",
		Code:         "BTC-USDT.CRYPTO",
	}
	q := CheckTradingDayQuery{Code: "BTC-USDT.CRYPTO", Date: "2024-06-01"}
	if err := ValidateCheckTradingDayData(q, td); err == nil {
		t.Fatal("expected .CRYPTO requires market CRYPTO")
	}
}

func TestValidateCheckTradingDayData_UnknownMarket(t *testing.T) {
	td := &TradingDayData{IsTradingDay: false, Date: "2024-06-01", Market: "NotKnown", Code: "FOO.BAR"}
	q := CheckTradingDayQuery{Code: "FOO.BAR", Date: "2024-06-01"}
	if err := ValidateCheckTradingDayData(q, td); err == nil {
		t.Fatal("expected unknown market error")
	}
}