package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// UpstreamMessage is one OpenAI-style chat message from GeeGooBot agent-api.
type UpstreamMessage struct {
	Role    string
	Content string
}

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

// NewUpstreamSession builds a session from agent-api message history.
func NewUpstreamSession(messages []UpstreamMessage) (*Session, string) {
	session := &Session{
		ID:        newSessionID(),
		CreatedAt: time.Now().UTC(),
	}
	var lastUser string
	for _, m := range messages {
		role := llm.Role(m.Role)
		if role != llm.RoleSystem && role != llm.RoleUser && role != llm.RoleAssistant {
			continue
		}
		session.AppendMessage(llm.Message{Role: role, Content: m.Content})
		if role == llm.RoleUser {
			lastUser = m.Content
		}
	}
	if len(session.Messages) == 0 {
		session.AppendMessage(llm.Message{Role: llm.RoleSystem, Content: chatSystemPrompt})
	}
	if len(session.Messages) > 0 {
		last := session.Messages[len(session.Messages)-1]
		if last.Role == llm.RoleUser && last.Content == lastUser {
			session.Messages = session.Messages[:len(session.Messages)-1]
		}
	}
	return session, lastUser
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
