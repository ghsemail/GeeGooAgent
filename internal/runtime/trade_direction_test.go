package runtime

import "testing"

func TestParseTradeDirection(t *testing.T) {
	cases := []struct {
		in   string
		want TradeDirection
	}{
		{"", TradeDirectionBoth},
		{"  ", TradeDirectionBoth},
		{"both", TradeDirectionBoth},
		{"BOTH", TradeDirectionBoth},
		{" long_only ", TradeDirectionLongOnly},
		{"LONG_ONLY", TradeDirectionLongOnly},
		{"short_only", TradeDirectionBoth},
		{"invalid", TradeDirectionBoth},
	}
	for _, c := range cases {
		if got := ParseTradeDirection(c.in); got != c.want {
			t.Fatalf("ParseTradeDirection(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestTradeDirectionLongOnlyEpisodeEval(t *testing.T) {
	if TradeDirectionBoth.LongOnlyEpisodeEval() {
		t.Fatal("both should not be long only")
	}
	if !TradeDirectionLongOnly.LongOnlyEpisodeEval() {
		t.Fatal("long_only should be long only")
	}
}
