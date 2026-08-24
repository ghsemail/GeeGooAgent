package runtimeapi

import "testing"

func TestRebindQuestionSQL(t *testing.T) {
	in := `SELECT * FROM t WHERE a = ? AND (b = ? OR c = '') LIMIT ?`
	want := `SELECT * FROM t WHERE a = $1 AND (b = $2 OR c = '') LIMIT $3`
	if got := rebindQuestionSQL(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
