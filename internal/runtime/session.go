package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// Session holds in-memory chat state for one conversation.
type Session struct {
	ID          string
	Messages    []llm.Message
	StepCounter int
	CreatedAt   time.Time
}

// NewSession creates a chat session with system prompt.
func NewSession() *Session {
	return &Session{
		ID: newSessionID(),
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: chatSystemPrompt},
		},
		CreatedAt: time.Now().UTC(),
	}
}

func newSessionID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "chat-" + hex.EncodeToString(b[:])
}

// AppendMessage adds a message to the session.
func (s *Session) AppendMessage(m llm.Message) {
	s.Messages = append(s.Messages, m)
}

// LLMMessages returns a copy for the gateway.
func (s *Session) LLMMessages() []llm.Message {
	out := make([]llm.Message, len(s.Messages))
	copy(out, s.Messages)
	return out
}
