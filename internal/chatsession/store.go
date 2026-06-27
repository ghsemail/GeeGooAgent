package chatsession

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// ChatStepRecord is one plan/tool/reply step in a persisted chat session.
type ChatStepRecord struct {
	Step       int       `json:"step"`
	Timestamp  time.Time `json:"timestamp"`
	Kind       string    `json:"kind"`
	ToolName   string    `json:"tool_name,omitempty"`
	ToolStatus string    `json:"tool_status,omitempty"`
	Summary    string    `json:"summary"`
}

// ChatSession is a persisted interactive chat session.
type ChatSession struct {
	ID          string        `json:"id"`
	Status      string        `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Messages    []llm.Message `json:"messages"`
	StepRecords []ChatStepRecord `json:"step_records"`
	StepCounter int           `json:"step_counter"`
}

// ChatSessionStore persists chat sessions under state/chat/{id}.
type ChatSessionStore struct {
	store *infra.StateStore
}

// NewChatSessionStore creates a store backed by FileStateStore root.
func NewChatSessionStore(store *infra.StateStore) *ChatSessionStore {
	return &ChatSessionStore{store: store}
}

func (s *ChatSessionStore) key(sessionID string) string {
	return "chat/" + sessionID
}

// Create allocates and saves a new active session with system prompt.
func (s *ChatSessionStore) Create() (*ChatSession, error) {
	session := &ChatSession{
		ID:        newChatSessionID(),
		Status:    "active",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: chatprompt.System()},
		},
	}
	if err := s.Save(session); err != nil {
		return nil, err
	}
	return session, nil
}

// Load reads a session by id.
func (s *ChatSessionStore) Load(sessionID string) (*ChatSession, error) {
	data, err := s.store.Load(s.key(sessionID))
	if err != nil || data == nil {
		return nil, err
	}
	return chatSessionFromMap(data)
}

// Save persists session state.
func (s *ChatSessionStore) Save(session *ChatSession) error {
	session.UpdatedAt = time.Now().UTC()
	return s.store.Save(s.key(session.ID), session.toMap())
}

// SyncChatSystemPrompt refreshes system message with tool activity summary.
func (c *ChatSession) SyncChatSystemPrompt() {
	content := chatprompt.System()
	if activity := c.ToolActivitySummary(); activity != "" {
		content = chatprompt.System() + "\n\n本会话 Tool 活动：\n" + activity
	}
	if len(c.Messages) > 0 && c.Messages[0].Role == llm.RoleSystem {
		c.Messages[0].Content = content
		return
	}
	c.Messages = append([]llm.Message{{Role: llm.RoleSystem, Content: content}}, c.Messages...)
}

// ToolActivitySummary lists market-related tools already called in this chat.
func (c *ChatSession) ToolActivitySummary() string {
	tracked := map[string]struct{}{
		"search_code": {}, "get_current_price": {}, "get_ticker": {},
		"get_mcp_analysis": {}, "get_capital_flow": {}, "get_capital_distribution": {},
	}
	var lines []string
	for _, msg := range c.Messages {
		if msg.Role != llm.RoleAssistant {
			continue
		}
		for _, call := range msg.ToolCalls {
			if _, ok := tracked[call.Name]; !ok {
				continue
			}
			parts := make([]string, 0, len(call.Arguments))
			for k, v := range call.Arguments {
				if v == nil || v == "" {
					continue
				}
				if m, ok := v.(map[string]any); ok && len(m) == 0 {
					continue
				}
				parts = append(parts, k+"="+formatAny(v))
			}
			lines = append(lines, "- "+call.Name+"("+joinParts(parts)+")")
		}
	}
	return joinLines(lines)
}

// SyncFromRuntime copies in-memory session state after a turn.
func (c *ChatSession) SyncFromRuntime(messages []llm.Message, stepCounter int, newRecords []ChatStepRecord) {
	c.Messages = make([]llm.Message, len(messages))
	copy(c.Messages, messages)
	c.StepCounter = stepCounter
	if len(newRecords) > 0 {
		c.StepRecords = append(c.StepRecords, newRecords...)
	}
}

// RuntimeMessages returns a copy of chat messages for the ReAct loop.
func (c *ChatSession) RuntimeMessages() []llm.Message {
	out := make([]llm.Message, len(c.Messages))
	copy(out, c.Messages)
	return out
}

func newChatSessionID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return "chat-" + hex.EncodeToString(b[:])
}

func (c *ChatSession) toMap() map[string]any {
	msgs := make([]map[string]any, 0, len(c.Messages))
	for _, m := range c.Messages {
		item := map[string]any{"role": string(m.Role), "content": m.Content}
		if m.ToolCallID != "" {
			item["tool_call_id"] = m.ToolCallID
		}
		if m.ReasoningContent != "" {
			item["reasoning_content"] = m.ReasoningContent
		}
		if len(m.ToolCalls) > 0 {
			calls := make([]map[string]any, 0, len(m.ToolCalls))
			for _, call := range m.ToolCalls {
				calls = append(calls, map[string]any{
					"id": call.ID, "name": call.Name, "arguments": call.Arguments,
				})
			}
			item["tool_calls"] = calls
		}
		msgs = append(msgs, item)
	}
	records := make([]map[string]any, 0, len(c.StepRecords))
	for _, rec := range c.StepRecords {
		records = append(records, map[string]any{
			"step": rec.Step, "timestamp": rec.Timestamp.Format(time.RFC3339),
			"kind": rec.Kind, "tool_name": rec.ToolName, "tool_status": rec.ToolStatus,
			"summary": rec.Summary,
		})
	}
	return map[string]any{
		"id": c.ID, "status": c.Status,
		"created_at": c.CreatedAt.Format(time.RFC3339),
		"updated_at": c.UpdatedAt.Format(time.RFC3339),
		"messages": msgs, "step_records": records, "step_counter": c.StepCounter,
	}
}

func chatSessionFromMap(data map[string]any) (*ChatSession, error) {
	session := &ChatSession{
		ID: stringField(data, "id"), Status: stringField(data, "status"),
		StepCounter: intField(data, "step_counter"),
	}
	if t, err := time.Parse(time.RFC3339, stringField(data, "created_at")); err == nil {
		session.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, stringField(data, "updated_at")); err == nil {
		session.UpdatedAt = t
	}
	if rawMsgs, ok := data["messages"].([]any); ok {
		for _, item := range rawMsgs {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			msg := llm.Message{Role: llm.Role(stringField(m, "role")), Content: stringField(m, "content")}
			msg.ToolCallID = stringField(m, "tool_call_id")
			msg.ReasoningContent = stringField(m, "reasoning_content")
			if calls, ok := m["tool_calls"].([]any); ok {
				for _, c := range calls {
					cm, ok := c.(map[string]any)
					if !ok {
						continue
					}
					args, _ := cm["arguments"].(map[string]any)
					msg.ToolCalls = append(msg.ToolCalls, llm.ToolCall{
						ID: stringField(cm, "id"), Name: stringField(cm, "name"), Arguments: args,
					})
				}
			}
			session.Messages = append(session.Messages, msg)
		}
	}
	if rawRecs, ok := data["step_records"].([]any); ok {
		for _, item := range rawRecs {
			rm, ok := item.(map[string]any)
			if !ok {
				continue
			}
			rec := ChatStepRecord{
				Step: intField(rm, "step"), Kind: stringField(rm, "kind"),
				ToolName: stringField(rm, "tool_name"), ToolStatus: stringField(rm, "tool_status"),
				Summary: stringField(rm, "summary"),
			}
			if t, err := time.Parse(time.RFC3339, stringField(rm, "timestamp")); err == nil {
				rec.Timestamp = t
			}
			session.StepRecords = append(session.StepRecords, rec)
		}
	}
	return session, nil
}

func stringField(m map[string]any, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}

func intField(m map[string]any, k string) int {
	switch v := m[k].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	default:
		return 0
	}
}

func formatAny(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		raw, _ := json.Marshal(t)
		return string(raw)
	}
}

func joinParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += ", " + parts[i]
	}
	return out
}

// ListSessionIDs returns chat session ids under chat/ prefix.
func (s *ChatSessionStore) ListSessionIDs() ([]string, error) {
	keys, err := s.store.ListKeys("chat")
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, key := range keys {
		if strings.HasPrefix(key, "chat/") {
			ids = append(ids, strings.TrimPrefix(key, "chat/"))
		}
	}
	return ids, nil
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	out := lines[0]
	for i := 1; i < len(lines); i++ {
		out += "\n" + lines[i]
	}
	return out
}
