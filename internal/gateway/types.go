// Package gateway is the IM Gateway (Hermes-style): platform adapters feed
// Agent.Run. Distinct from internal/llm.Gateway (model channel).
package gateway

import "time"

// Platform identifies an IM backend.
type Platform string

const (
	PlatformFeishu Platform = "feishu"
)

// ChatType classifies the inbound conversation.
type ChatType string

const (
	ChatTypeDM    ChatType = "dm"
	ChatTypeGroup ChatType = "group"
)

// InboundEvent is a normalized message from any platform adapter.
type InboundEvent struct {
	Platform  Platform
	ChatID    string
	ChatType  ChatType
	UserID    string // preferred stable id (union_id or open_id)
	UserName  string
	MessageID string
	Text      string
	Mentioned bool // true when bot was @mentioned (always true for DM)
	Raw       any  // optional platform payload for debugging
	Received  time.Time
}

// SessionKey builds the persistent routing key (group sessions per user).
func SessionKey(p Platform, chatID, userID string) string {
	return string(p) + ":" + chatID + ":u:" + userID
}

// OutboundText is a plain-text reply (M1). Media/cards reserved for M2+.
type OutboundText struct {
	ChatID    string
	Text      string
	ReplyToID string // optional parent message id
}

// HomeChannel is where cron/notifications are delivered (M2).
type HomeChannel struct {
	Platform Platform
	ChatID   string
	Name     string
}
