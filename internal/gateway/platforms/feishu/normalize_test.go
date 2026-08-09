package feishu

import (
	"testing"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"

	"github.com/ghsemail/GeeGooAgent/internal/gateway"
)

func strPtr(s string) *string { return &s }

func TestNormalizeDMText(t *testing.T) {
	content := `{"text":"hello agent"}`
	ev := &larkim.P2MessageReceiveV1{
		Event: &larkim.P2MessageReceiveV1Data{
			Sender: &larkim.EventSender{
				SenderType: strPtr("user"),
				SenderId:   &larkim.UserId{OpenId: strPtr("ou_alice")},
			},
			Message: &larkim.EventMessage{
				MessageId:   strPtr("om_1"),
				ChatId:      strPtr("oc_dm"),
				ChatType:    strPtr("p2p"),
				MessageType: strPtr("text"),
				Content:     &content,
			},
		},
	}
	got, ok := NormalizeMessage(ev, "ou_bot", "Bot", true, "allowlist", nil)
	if !ok {
		t.Fatal("expected ok")
	}
	if got.ChatType != gateway.ChatTypeDM || got.Text != "hello agent" || got.UserID != "ou_alice" {
		t.Fatalf("%+v", got)
	}
	if !got.Mentioned {
		t.Fatal("DM should be mentioned=true")
	}
}

func TestNormalizeGroupRequiresMention(t *testing.T) {
	content := `{"text":"hello"}`
	ev := &larkim.P2MessageReceiveV1{
		Event: &larkim.P2MessageReceiveV1Data{
			Sender: &larkim.EventSender{
				SenderType: strPtr("user"),
				SenderId:   &larkim.UserId{OpenId: strPtr("ou_alice")},
			},
			Message: &larkim.EventMessage{
				MessageId:   strPtr("om_2"),
				ChatId:      strPtr("oc_group"),
				ChatType:    strPtr("group"),
				MessageType: strPtr("text"),
				Content:     &content,
			},
		},
	}
	_, ok := NormalizeMessage(ev, "ou_bot", "Bot", true, "open", nil)
	if ok {
		t.Fatal("group without @ should be ignored")
	}
}

func TestNormalizeGroupWithMention(t *testing.T) {
	content := `{"text":"@_user_1 please check"}`
	ev := &larkim.P2MessageReceiveV1{
		Event: &larkim.P2MessageReceiveV1Data{
			Sender: &larkim.EventSender{
				SenderType: strPtr("user"),
				SenderId:   &larkim.UserId{OpenId: strPtr("ou_alice")},
			},
			Message: &larkim.EventMessage{
				MessageId:   strPtr("om_3"),
				ChatId:      strPtr("oc_group"),
				ChatType:    strPtr("group"),
				MessageType: strPtr("text"),
				Content:     &content,
				Mentions: []*larkim.MentionEvent{{
					Key:           strPtr("@_user_1"),
					MentionedType: strPtr("bot"),
					Id:            &larkim.UserId{OpenId: strPtr("ou_bot")},
					Name:          strPtr("Bot"),
				}},
			},
		},
	}
	got, ok := NormalizeMessage(ev, "ou_bot", "Bot", true, "open", nil)
	if !ok {
		t.Fatal("expected ok")
	}
	if got.Text != "please check" {
		t.Fatalf("text=%q", got.Text)
	}
}

func TestNormalizeGroupAllowlist(t *testing.T) {
	content := `{"text":"@_user_1 hi"}`
	ev := &larkim.P2MessageReceiveV1{
		Event: &larkim.P2MessageReceiveV1Data{
			Sender: &larkim.EventSender{
				SenderType: strPtr("user"),
				SenderId:   &larkim.UserId{OpenId: strPtr("ou_bob")},
			},
			Message: &larkim.EventMessage{
				MessageId:   strPtr("om_4"),
				ChatId:      strPtr("oc_group"),
				ChatType:    strPtr("group"),
				MessageType: strPtr("text"),
				Content:     &content,
				Mentions: []*larkim.MentionEvent{{
					Key:           strPtr("@_user_1"),
					MentionedType: strPtr("bot"),
					Id:            &larkim.UserId{OpenId: strPtr("ou_bot")},
				}},
			},
		},
	}
	allowed := map[string]struct{}{"ou_alice": {}}
	_, ok := NormalizeMessage(ev, "ou_bot", "Bot", true, "allowlist", allowed)
	if ok {
		t.Fatal("bob not on allowlist")
	}
}

func TestNormalizeIgnoresBotSender(t *testing.T) {
	content := `{"text":"ping"}`
	ev := &larkim.P2MessageReceiveV1{
		Event: &larkim.P2MessageReceiveV1Data{
			Sender: &larkim.EventSender{
				SenderType: strPtr("bot"),
				SenderId:   &larkim.UserId{OpenId: strPtr("ou_otherbot")},
			},
			Message: &larkim.EventMessage{
				MessageId:   strPtr("om_5"),
				ChatId:      strPtr("oc_dm"),
				ChatType:    strPtr("p2p"),
				MessageType: strPtr("text"),
				Content:     &content,
			},
		},
	}
	_, ok := NormalizeMessage(ev, "ou_bot", "Bot", true, "open", nil)
	if ok {
		t.Fatal("bot sender should be ignored")
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	cfg := LoadConfigFromEnv(func(k string) string {
		switch k {
		case "FEISHU_APP_ID":
			return "cli_x"
		case "FEISHU_APP_SECRET":
			return "sec"
		case "FEISHU_ALLOWED_USERS":
			return "ou_a, ou_b"
		case "FEISHU_REQUIRE_MENTION":
			return "false"
		default:
			return ""
		}
	})
	if len(cfg.AllowedUsers) != 2 || cfg.RequireMention {
		t.Fatalf("%+v", cfg)
	}
	ad := NewAdapter(cfg)
	if !ad.Configured() {
		t.Fatal("expected configured")
	}
}
