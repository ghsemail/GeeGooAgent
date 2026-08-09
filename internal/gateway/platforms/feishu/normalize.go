package feishu

import (
	"encoding/json"
	"strings"
	"time"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"

	"github.com/ghsemail/GeeGooAgent/internal/gateway"
)

// NormalizeMessage converts a Feishu receive-message event into InboundEvent.
// Returns ok=false when the event should be ignored (empty, bot sender, etc.).
func NormalizeMessage(event *larkim.P2MessageReceiveV1, botOpenID, botName string, requireMention bool, groupPolicy string, allowed map[string]struct{}) (gateway.InboundEvent, bool) {
	if event == nil || event.Event == nil || event.Event.Message == nil {
		return gateway.InboundEvent{}, false
	}
	msg := event.Event.Message
	sender := event.Event.Sender
	if sender != nil && sender.SenderType != nil && strings.EqualFold(*sender.SenderType, "bot") {
		return gateway.InboundEvent{}, false
	}

	chatType := gateway.ChatTypeGroup
	if msg.ChatType != nil && strings.EqualFold(*msg.ChatType, "p2p") {
		chatType = gateway.ChatTypeDM
	}

	userID, userName := senderIdentity(sender)
	if userID == "" {
		return gateway.InboundEvent{}, false
	}

	mentioned := chatType == gateway.ChatTypeDM || mentionedBot(msg.Mentions, botOpenID, botName)
	if chatType == gateway.ChatTypeGroup {
		switch strings.ToLower(groupPolicy) {
		case "disabled":
			return gateway.InboundEvent{}, false
		case "allowlist":
			if len(allowed) > 0 {
				if _, ok := allowed[userID]; !ok {
					return gateway.InboundEvent{}, false
				}
			}
			if requireMention && !mentioned {
				return gateway.InboundEvent{}, false
			}
		default: // open
			if requireMention && !mentioned {
				return gateway.InboundEvent{}, false
			}
		}
	}

	text := extractText(msg)
	text = stripMentionTokens(text, msg.Mentions)
	text = strings.TrimSpace(text)
	if text == "" {
		return gateway.InboundEvent{}, false
	}

	chatID := deref(msg.ChatId)
	msgID := deref(msg.MessageId)
	return gateway.InboundEvent{
		Platform:  gateway.PlatformFeishu,
		ChatID:    chatID,
		ChatType:  chatType,
		UserID:    userID,
		UserName:  userName,
		MessageID: msgID,
		Text:      text,
		Mentioned: mentioned,
		Received:  time.Now().UTC(),
	}, true
}

func senderIdentity(sender *larkim.EventSender) (id, name string) {
	if sender == nil || sender.SenderId == nil {
		return "", ""
	}
	uid := sender.SenderId
	// Prefer open_id so FEISHU_ALLOWED_USERS matches Feishu console ids.
	if uid.OpenId != nil && strings.TrimSpace(*uid.OpenId) != "" {
		id = strings.TrimSpace(*uid.OpenId)
	} else if uid.UnionId != nil && strings.TrimSpace(*uid.UnionId) != "" {
		id = strings.TrimSpace(*uid.UnionId)
	} else if uid.UserId != nil {
		id = strings.TrimSpace(*uid.UserId)
	}
	return id, ""
}

func mentionedBot(mentions []*larkim.MentionEvent, botOpenID, botName string) bool {
	for _, m := range mentions {
		if m == nil {
			continue
		}
		if m.MentionedType != nil && strings.EqualFold(*m.MentionedType, "bot") {
			if botOpenID == "" && botName == "" {
				return true
			}
			if m.Id != nil && m.Id.OpenId != nil && botOpenID != "" && *m.Id.OpenId == botOpenID {
				return true
			}
			if m.Name != nil && botName != "" && strings.EqualFold(*m.Name, botName) {
				return true
			}
			if botOpenID == "" {
				return true
			}
		}
		// @all
		if m.Key != nil && (*m.Key == "@_all" || strings.Contains(*m.Key, "all")) {
			return true
		}
	}
	return false
}

func extractText(msg *larkim.EventMessage) string {
	if msg == nil || msg.Content == nil {
		return ""
	}
	raw := strings.TrimSpace(*msg.Content)
	msgType := ""
	if msg.MessageType != nil {
		msgType = *msg.MessageType
	}
	switch msgType {
	case "text", "":
		var payload struct {
			Text string `json:"text"`
		}
		if json.Unmarshal([]byte(raw), &payload) == nil && payload.Text != "" {
			return payload.Text
		}
		return raw
	case "post":
		return flattenPost(raw)
	default:
		// M1: ignore non-text media as empty (no agent turn).
		return ""
	}
}

func flattenPost(raw string) string {
	var root map[string]any
	if json.Unmarshal([]byte(raw), &root) != nil {
		return raw
	}
	var parts []string
	for _, locale := range root {
		loc, ok := locale.(map[string]any)
		if !ok {
			continue
		}
		content, _ := loc["content"].([]any)
		for _, row := range content {
			cells, _ := row.([]any)
			var line []string
			for _, cell := range cells {
				m, _ := cell.(map[string]any)
				if m == nil {
					continue
				}
				if t, ok := m["text"].(string); ok && t != "" {
					line = append(line, t)
				}
			}
			if len(line) > 0 {
				parts = append(parts, strings.Join(line, ""))
			}
		}
		if len(parts) > 0 {
			break
		}
	}
	return strings.Join(parts, "\n")
}

func stripMentionTokens(text string, mentions []*larkim.MentionEvent) string {
	out := text
	for _, m := range mentions {
		if m == nil || m.Key == nil {
			continue
		}
		out = strings.ReplaceAll(out, *m.Key, "")
		if m.Name != nil && *m.Name != "" {
			out = strings.ReplaceAll(out, "@"+*m.Name, "")
		}
	}
	return strings.TrimSpace(out)
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
