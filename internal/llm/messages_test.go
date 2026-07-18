package llm

import "testing"

func TestSanitizeMessagesMergesConsecutiveUsers(t *testing.T) {
	in := []Message{
		{Role: RoleUser, Content: "a"},
		{Role: RoleUser, Content: "b"},
	}
	out := SanitizeMessages(in)
	if len(out) != 1 || out[0].Content != "a\n\nb" {
		t.Fatalf("got %+v", out)
	}
}
