package chatprompt_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
)

func TestReplyFormatReminder(t *testing.T) {
	if chatprompt.ReplyFormatReminder(0) != "" {
		t.Fatal("round 0 should not inject reminder")
	}
	got := chatprompt.ReplyFormatReminder(1)
	if got == "" {
		t.Fatal("round 1 should inject reminder")
	}
	if !strings.Contains(got, "[FORMAT]") {
		t.Fatalf("reminder should mention FORMAT tag: %q", got)
	}
}
