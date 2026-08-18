package stockfmt

import (
	"testing"
)

func TestExtractWeeklyKeyLevelsRange(t *testing.T) {
	text := `### 关键点位
- 上方压力：**450.4 / 455~460 / 478~485**
- 下方支撑：**436~440 / 432 / 420**`
	levels := ExtractWeeklyKeyLevels(text, "00700.HK")
	if !levels.Valid {
		t.Fatalf("expected valid levels, warnings=%v", levels.Warnings)
	}
	if *levels.Support != 436 {
		t.Fatalf("support=%v want 436", *levels.Support)
	}
	if *levels.Resistance != 460 {
		t.Fatalf("resistance=%v want 460", *levels.Resistance)
	}
}

func TestExtractWeeklyKeyLevelsRejectsCodeArtifact(t *testing.T) {
	levels := ExtractWeeklyKeyLevels("支撑位 700\n阻力位 700", "00700.HK")
	if levels.Valid {
		t.Fatalf("should reject code artifact levels: %+v", levels)
	}
}

func TestApplyPriceSanity(t *testing.T) {
	s, r := 436.0, 460.0
	levels := ApplyPriceSanity(KeyLevels{Support: &s, Resistance: &r, Valid: true}, 442.0)
	if !levels.Valid {
		t.Fatalf("expected sane levels: %+v", levels)
	}
	levels = ApplyPriceSanity(levels, 50)
	if levels.Valid {
		t.Fatal("expected invalid after sanity check")
	}
}

func TestCapitalFlowDivergent(t *testing.T) {
	s := "小单净流入5.46亿 · 大单净流出5668.08万 · 简要解读：小单净流入而主力偏弱"
	if !CapitalFlowDivergent(s) {
		t.Fatal("expected divergent capital flow")
	}
}
