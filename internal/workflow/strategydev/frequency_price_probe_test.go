package strategydev

import "testing"

func TestPriceAnalysisProbeArgsOmitsLimitUsesMonthsBack(t *testing.T) {
	args := PriceAnalysisProbeArgs("00700.HK", "5m", 3)
	if _, ok := args["limit"]; ok {
		t.Fatalf("limit must be omitted for price analysis probe: %v", args)
	}
	if args["months_back"] != 3 {
		t.Fatalf("months_back=%v want 3", args["months_back"])
	}
	if args["frequency"] != "5m" {
		t.Fatalf("frequency=%v", args["frequency"])
	}
	if args["code"] != "00700.HK" {
		t.Fatalf("code=%v", args["code"])
	}
}

func TestPriceAnalysisProbeArgsCapsMonthsBack(t *testing.T) {
	args := PriceAnalysisProbeArgs("AAPL.US", "60m", 12)
	if args["months_back"] != 6 {
		t.Fatalf("months_back=%v want cap 6", args["months_back"])
	}
}
