package slots

import "testing"

func TestKlinePatternSummary_SingleBull(t *testing.T) {
	s := KlinePatternSummary([]KlinePatternHit{{Name: "Hammer", Direction: 1, LabelCN: "锤头线"}})
	if s != "锤头线(看多)" {
		t.Fatalf("got %q", s)
	}
}
