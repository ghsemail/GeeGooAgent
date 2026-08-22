package chatsession_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
)

func TestActiveScopesFromSession(t *testing.T) {
	s := &chatsession.ChatSession{
		Metadata: map[string]any{
			"active_scopes":    []string{"stock:00700.HK"},
			"context_profiles": []string{"market:HK"},
		},
	}
	got := chatsession.ActiveScopesFromSession(s)
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
}

func TestSetActiveScopesWritesBothKeys(t *testing.T) {
	s := &chatsession.ChatSession{Metadata: map[string]any{}}
	chatsession.SetActiveScopes(s, []string{"automation:b1", "bot:b2"})
	active, ok := s.Metadata["active_scopes"].([]string)
	if !ok || len(active) != 2 {
		t.Fatalf("active_scopes=%v", s.Metadata["active_scopes"])
	}
	legacy, ok := s.Metadata["context_profiles"].([]string)
	if !ok || len(legacy) != 2 {
		t.Fatalf("context_profiles=%v", s.Metadata["context_profiles"])
	}
}
