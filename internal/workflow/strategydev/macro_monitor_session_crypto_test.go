package strategydev

import (
	"testing"
	"time"
)

// CRYPTO uses vendor label days; session gate is not US 16:00 or UTC 23:59 wall clock.
func TestSessionCloseInstant_CryptoVendorNotUS16ET(t *testing.T) {
	crypto, ok := macroSessionSpecForCode("BTC-USD.CRYPTO")
	if !ok || !crypto.cryptoVendorLabel {
		t.Fatal("crypto vendor spec")
	}
	us, ok := macroSessionSpecForCode("SPY.US")
	if !ok {
		t.Fatal("us spec")
	}
	day := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	cClose := sessionCloseInstant(day, crypto)
	uClose := sessionCloseInstant(day, us)
	if cClose.Equal(uClose) {
		t.Fatalf("CRYPTO metadata close %v must differ from US %v", cClose, uClose)
	}
	if uClose.Hour() != 16 {
		t.Fatalf("us close hour %d want 16", uClose.Hour())
	}
	if sessionDayComplete(day, day, crypto) && !crypto.cryptoVendorLabel {
		t.Fatal("vendor mode should not use equity sessionDayComplete")
	}
}