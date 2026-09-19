package diagram

import "testing"

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"check_trading_day": "check_trading_day",
		"index_000001.SS":   "index_000001_SS",
		"1start":            "n1start",
		"":                  "node",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Fatalf("Slug(%q)=%q want %q", in, got, want)
		}
	}
}
