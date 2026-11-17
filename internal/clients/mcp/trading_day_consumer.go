package mcp

import (
	"fmt"
	"strings"
	"time"
)

// CheckTradingDayConsumerContractVersion is the Agent-side validator for daily_macro calendar
// consumption. Bump when required response fields or echo rules change (App contract).
const CheckTradingDayConsumerContractVersion = 2

// CheckTradingDayQuery is what Agent sends today (GeeGooBot may ignore date until App ships).
type CheckTradingDayQuery struct {
	Code string // symbol, e.g. 000300.SH, BTC-USD.CRYPTO
	Date string // YYYY-MM-DD calendar day under test (per-bar for macro cutoff)
}

// ValidateCheckTradingDayData fail-closed unless the response clearly answers the query.
// Legacy Bot always echoes "today" in data.date — mismatch against query.Date must error.
// ValidCheckTradingDayDate is true for App contract YYYY-MM-DD (calendar query).
func ValidCheckTradingDayDate(date string) bool {
	date = strings.TrimSpace(date)
	if date == "" {
		return false
	}
	_, err := time.Parse("2006-01-02", date)
	return err == nil
}

// ValidateCheckTradingDayData fail-closed for daily_macro when query carries a legal date.
func ValidateCheckTradingDayData(q CheckTradingDayQuery, td *TradingDayData) error {
	if td == nil {
		return fmt.Errorf("checkTradingDay: empty data")
	}
	code := strings.TrimSpace(q.Code)
	if code == "" {
		return fmt.Errorf("checkTradingDay: empty query code")
	}
	day := strings.TrimSpace(q.Date)
	if day == "" {
		return fmt.Errorf("checkTradingDay: empty query date")
	}
	if !ValidCheckTradingDayDate(day) {
		return fmt.Errorf("checkTradingDay: invalid query date %q", day)
	}
	if strings.TrimSpace(td.Code) != "" && !strings.EqualFold(strings.TrimSpace(td.Code), code) {
		return fmt.Errorf("checkTradingDay: code echo %q != query %q", td.Code, code)
	}
	respDay := strings.TrimSpace(td.Date)
	if respDay == "" {
		return fmt.Errorf("checkTradingDay: missing data.date echo")
	}
	if respDay != day {
		return fmt.Errorf("checkTradingDay: date echo %q != queried %q (stale today-only Bot?)", respDay, day)
	}
	mkt := strings.TrimSpace(td.Market)
	if mkt == "" || strings.EqualFold(mkt, "NotKnown") {
		return fmt.Errorf("checkTradingDay: missing or unknown market echo")
	}
	if err := validateTradingDayMarketForCode(code, mkt); err != nil {
		return err
	}
	return nil
}

func validateTradingDayMarketForCode(code, market string) error {
	u := strings.ToUpper(strings.TrimSpace(code))
	m := strings.ToUpper(strings.TrimSpace(market))
	if strings.HasSuffix(u, ".CRYPTO") {
		if m != "CRYPTO" {
			return fmt.Errorf("checkTradingDay: .CRYPTO code requires market CRYPTO, got %q", market)
		}
		return nil
	}
	switch {
	case strings.HasSuffix(u, ".US") || strings.HasSuffix(u, ".NYB"):
		if m != "US" {
			return fmt.Errorf("checkTradingDay: US symbol market echo %q", market)
		}
	case strings.HasSuffix(u, ".HK"):
		if m != "HK" {
			return fmt.Errorf("checkTradingDay: HK symbol market echo %q", market)
		}
	case strings.HasSuffix(u, ".SH") || strings.HasSuffix(u, ".SZ") || strings.HasSuffix(u, ".SS"):
		if m != "CN" {
			return fmt.Errorf("checkTradingDay: CN symbol market echo %q", market)
		}
	}
	return nil
}