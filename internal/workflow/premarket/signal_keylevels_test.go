package premarket

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
)

func TestKeyLevelsFromSupportingPrice(t *testing.T) {
	resp := &mcp.SupportingPriceResponse{
		Code: "00700.HK",
		Data: mcp.SupportingPriceData{
			Judgment: mcp.SupportingPriceJudgment{
				Support: &mcp.SupportingPriceSideJudgment{
					Center: 438, Low: 436, High: 440, Sources: []string{"daily_swing_low", "daily_val"},
				},
				Resistance: &mcp.SupportingPriceSideJudgment{
					Center: 458, Low: 456, High: 460, Sources: []string{"daily_swing_high"},
				},
			},
			Legacy: &mcp.SupportingPriceLegacy{QFLSupport: 438, QFLResistance: 458},
		},
	}
	levels := keyLevelsFromSupportingPrice(resp)
	if !levels.Valid {
		t.Fatal("expected valid levels")
	}
	if *levels.Support != 438 || *levels.Resistance != 458 {
		t.Fatalf("centers: support=%v resistance=%v", *levels.Support, *levels.Resistance)
	}
	if levels.Provenance.SupportZone != "436.00~440.00" {
		t.Fatalf("zone: %s", levels.Provenance.SupportZone)
	}
}
