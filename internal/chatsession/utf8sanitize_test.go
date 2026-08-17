package chatsession

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func TestValidUTF8ReplacesInvalidSequence(t *testing.T) {
	// 0xe6 0x9c 0x2e is invalid UTF-8 (truncated CJK + ASCII period).
	bad := string([]byte{0xe6, 0x9c, 0x2e})
	got := ValidUTF8(bad)
	if !utf8.ValidString(got) {
		t.Fatalf("not valid utf8: %q", got)
	}
	if strings.Contains(got, bad) {
		t.Fatalf("expected replacement, got %q", got)
	}
}

func TestSanitizeForPersistMessages(t *testing.T) {
	bad := string([]byte{0xe6, 0x9c, 0x2e})
	s := &ChatSession{
		Title: bad,
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: "腾讯" + bad},
			{Role: llm.RoleAssistant, Content: "ok"},
		},
	}
	s.SanitizeForPersist()
	if !utf8.ValidString(s.Title) || !utf8.ValidString(s.Messages[0].Content) {
		t.Fatalf("sanitize failed: title=%q content=%q", s.Title, s.Messages[0].Content)
	}
}
