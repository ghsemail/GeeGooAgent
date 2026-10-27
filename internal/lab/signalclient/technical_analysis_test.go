package signalclient

import "testing"

func TestNormalizeStrategyVersion(t *testing.T) {
	cases := map[string]string{
		"": "alpha", "alpha": "alpha", "Alpha": "alpha", "a": "alpha", "A": "alpha", "α": "alpha",
		"beta": "beta",
	}
	for in, want := range cases {
		if got := NormalizeStrategyVersion(in); got != want {
			t.Fatalf("NormalizeStrategyVersion(%q)=%q want %q", in, got, want)
		}
	}
}
