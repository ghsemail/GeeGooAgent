package llm_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func TestFailoverEligible(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		code int
		want bool
	}{
		{401, true},
		{403, true},
		{429, true},
		{500, true},
		{502, true},
		{400, false},
		{404, false},
	} {
		err := &llm.HTTPError{StatusCode: tc.code, Body: "x"}
		if got := llm.FailoverEligible(err); got != tc.want {
			t.Fatalf("code %d: got %v want %v", tc.code, got, tc.want)
		}
	}
	if llm.FailoverEligible(nil) {
		t.Fatal("nil should not failover")
	}
}
