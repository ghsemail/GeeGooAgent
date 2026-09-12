package runtimeapi

import "testing"

func TestEvalRunVisibleToUser(t *testing.T) {
	tests := []struct {
		row, req string
		want     bool
	}{
		{"", "", true},
		{"alice", "", false},
		{"", "alice", true},
		{"alice", "alice", true},
		{"bob", "alice", false},
		{"alice", "bob", false},
	}
	for _, tc := range tests {
		if got := evalRunVisibleToUser(tc.row, tc.req); got != tc.want {
			t.Fatalf("evalRunVisibleToUser(%q,%q)=%v want %v", tc.row, tc.req, got, tc.want)
		}
	}
}

func TestNormalizeEvalRunStatus(t *testing.T) {
	if normalizeEvalRunStatus("pass") != "passed" {
		t.Fatal("pass")
	}
	if normalizeEvalRunStatus("fail") != "failed" {
		t.Fatal("fail")
	}
	if normalizeEvalRunStatus("passed") != "passed" {
		t.Fatal("passed")
	}
}
