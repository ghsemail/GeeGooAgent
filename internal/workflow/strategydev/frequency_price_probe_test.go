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

func TestPriceAnalysisProbeArgsWeeklyAllowsExtendedMonthsBack(t *testing.T) {
	args := PriceAnalysisProbeArgs("AAPL.US", "weekly", 10)
	if args["months_back"] != 10 {
		t.Fatalf("weekly months_back=%v want 10 (coarse TF fetch cap)", args["months_back"])
	}
	args = PriceAnalysisProbeArgs("AAPL.US", "weekly", 20)
	if args["months_back"] != 12 {
		t.Fatalf("weekly months_back=%v want hard cap 12", args["months_back"])
	}
}
