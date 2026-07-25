package facts

import "testing"

func TestBuildFTSQuery(t *testing.T) {
	if got := buildFTSQuery("when am I meeting Alex?"); got != "when | am | meeting | alex" {
		t.Fatalf("unexpected fts query: %q", got)
	}
	if buildFTSQuery("a") != "" {
		t.Fatal("expected empty for short tokens")
	}
}

func TestFormatAndSplit(t *testing.T) {
	sub, body := splitBracketSubject("[alex] prefers mornings")
	if sub != "alex" || body != "prefers mornings" {
		t.Fatalf("split failed: %q %q", sub, body)
	}
	if Format("alex", "prefers mornings") != "[alex] prefers mornings" {
		t.Fatal("format mismatch")
	}
}

func TestNormalizeSubject(t *testing.T) {
	if normalizeSubject("Alex") != "alex" {
		t.Fatal("expected lowercase subject")
	}
}
