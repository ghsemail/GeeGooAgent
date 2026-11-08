package strategydev

import "testing"

func TestLabTAFrequenciesJEVStandard(t *testing.T) {
	got := labTAFrequencies(LabDetailInput{SignalFreq: "5m"})
	if len(got) != 2 || got[0] != "daily" || got[1] != "60m" {
		t.Fatalf("labTAFrequencies(5m)=%v want [daily 60m]", got)
	}
	got = labTAFrequencies(LabDetailInput{SignalFreq: "60m"})
	if len(got) != 2 || got[0] != "daily" || got[1] != "60m" {
		t.Fatalf("labTAFrequencies(60m)=%v want [daily 60m]", got)
	}
	got = labTAFrequencies(LabDetailInput{
		SignalFreq:      "60m",
		KlineJEVProfile: LabKlineJEVProfileDailyMacro,
	})
	if len(got) != 1 || got[0] != "daily" {
		t.Fatalf("labTAFrequencies(daily_macro)=%v want [daily]", got)
	}
}
