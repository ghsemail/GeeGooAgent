package newsrunner

import "testing"

func TestPadHKTicker(t *testing.T) {
	if got := padHKTicker("700"); got != "00700" {
		t.Fatalf("padHKTicker(700)=%q want 00700", got)
	}
	if got := padHKTicker("00700"); got != "00700" {
		t.Fatalf("padHKTicker(00700)=%q want 00700", got)
	}
}

func TestHKADRMap(t *testing.T) {
	if hkADR("700") != "TCEHY" {
		t.Fatalf("hkADR(700)=%q", hkADR("700"))
	}
	if hkStockName("700") != "腾讯" {
		t.Fatalf("hkStockName(700)=%q", hkStockName("700"))
	}
}

func TestParseStockCodeHK(t *testing.T) {
	market, ticker := parseStockCode("00700.HK")
	if market != "HK" || ticker != "700" {
		t.Fatalf("parseStockCode(00700.HK)=%q,%q", market, ticker)
	}
}
