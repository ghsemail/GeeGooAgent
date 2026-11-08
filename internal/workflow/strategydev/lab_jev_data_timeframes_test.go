package strategydev

import "testing"

func TestLabTAFrequenciesJEVStandard(t *testing.T) {
	got := labTAFrequencies("5m")
	if len(got) != 2 || got[0] != "daily" || got[1] != "60m" {
		t.Fatalf("labTAFrequencies(5m)=%v want [daily 60m]", got)
	}
	got = labTAFrequencies("60m")
	if len(got) != 2 || got[0] != "daily" || got[1] != "60m" {
		t.Fatalf("labTAFrequencies(60m)=%v want [daily 60m]", got)
	}
}
