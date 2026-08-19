package usernotice

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/gateway/feishustore"
)

func TestSectionToCreds_roundTrip(t *testing.T) {
	in := FeishuSection{
		Connected:    true,
		AppID:        "cli_test",
		AppSecret:    "secret",
		Domain:       "feishu",
		MCPToken:     "mcp_tok",
		ReceiveID:    "ou_abc",
		BotName:      "GeeGoo助手",
		BotOpenID:    "bot_oid",
		AllowedUsers: []string{"ou_abc"},
	}
	c := sectionToCreds("6366170502d5c175fd586fe8", in)
	if c == nil || !c.Configured() {
		t.Fatal("expected configured creds")
	}
	out := credsToSection(c)
	if out.AppID != in.AppID || out.AppSecret != in.AppSecret {
		t.Fatalf("secrets mismatch: %+v", out)
	}
	if out.ReceiveID != in.ReceiveID || !out.Connected {
		t.Fatalf("routing mismatch: %+v", out)
	}
}

func TestCredsToSection_disabled(t *testing.T) {
	c := &feishustore.Creds{
		UserID:    "u1",
		AppID:     "cli_x",
		AppSecret: "sec",
		Enabled:   false,
	}
	s := credsToSection(c)
	if s.Connected {
		t.Fatal("disabled creds should not be connected")
	}
}
