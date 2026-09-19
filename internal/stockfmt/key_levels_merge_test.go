package stockfmt

import "testing"

func TestMergeKeyLevelsPrefersEngine(t *testing.T) {
	s, r := 440.0, 460.0
	engine := KeyLevels{
		Support: &s, Resistance: &r, Valid: true,
		Provenance: KeyLevelProvenance{EngineUsed: true, SupportZone: "438~442", ResistZone: "458~462"},
	}
	ts, tr := 436.0, 455.0
	text := KeyLevels{Support: &ts, Resistance: &tr, Valid: true}
	out := MergeKeyLevels(engine, text, 450)
	if !out.Valid || *out.Support != 440 || *out.Resistance != 460 {
		t.Fatalf("engine levels expected, got %+v", out)
	}
	if !out.Provenance.Merged {
		t.Fatalf("expected merged provenance")
	}
}

func TestMergeKeyLevelsFallbackText(t *testing.T) {
	ts, tr := 436.0, 460.0
	text := KeyLevels{Support: &ts, Resistance: &tr, Valid: true}
	out := MergeKeyLevels(KeyLevels{}, text, 442)
	if !out.Valid || *out.Support != 436 {
		t.Fatalf("text fallback expected, got %+v", out)
	}
}
