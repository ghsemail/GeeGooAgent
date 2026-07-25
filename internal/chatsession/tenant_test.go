package chatsession

import "testing"

func TestFilterEntriesByUser(t *testing.T) {
	entries := []ChatSessionIndexEntry{
		{ID: "a", Metadata: map[string]any{"user_id": "u1"}},
		{ID: "b", Metadata: map[string]any{"user_id": "u2"}},
		{ID: "c", Metadata: map[string]any{}},
	}
	got := FilterEntriesByUser(entries, "u1")
	if len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("filter u1: %+v", got)
	}
	if len(FilterEntriesByUser(entries, "")) != 3 {
		t.Fatal("empty user should return all")
	}
}

func TestEnforceAccessClaimLegacy(t *testing.T) {
	s := &ChatSession{ID: "legacy", Metadata: map[string]any{}}
	if !EnforceAccess(s, "u9") {
		t.Fatal("expected claim")
	}
	if UserIDFromSession(s) != "u9" {
		t.Fatalf("owner = %q", UserIDFromSession(s))
	}
	if EnforceAccess(s, "other") {
		t.Fatal("other user should be denied")
	}
}
