package chatsession

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sort"
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
	ID          string           `json:"id"`
	Title       string           `json:"title,omitempty"`
	Tags        []string         `json:"tags,omitempty"`
	Summary     string           `json:"summary,omitempty"`
	ToolNames   []string         `json:"tool_names,omitempty"`
	Metadata    map[string]any   `json:"metadata,omitempty"`
	Status      string           `json:"status"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	Messages    []llm.Message    `json:"messages"`
	StepRecords []ChatStepRecord `json:"step_records"`
	StepCounter int              `json:"step_counter"`
}

// ChatSessionIndexEntry is a compact manifest row for session lookup.
type ChatSessionIndexEntry struct {
	ID           string
	Title        string
	Tags         []string
	Summary      string
	ToolNames    []string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	MessageCount int
	StepCount    int
	Metadata     map[string]any
}

// ChatSessionIndex stores searchable session metadata.
type ChatSessionIndex struct {
	Version   int
	UpdatedAt time.Time
	Sessions  []ChatSessionIndexEntry
}

// SessionStore is the persistence abstraction for chat sessions.
// Both the legacy file-backed store and the SQLite store implement it.
type SessionStore interface {
	Create() (*ChatSession, error)
	Load(sessionID string) (*ChatSession, error)
	Save(session *ChatSession) error
	ListIndexedSessions() ([]ChatSessionIndexEntry, error)
	ListSessionIDs() ([]string, error)
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

func (s *ChatSessionStore) indexKey() string {
	return "chat_index"
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
	session, err := chatSessionFromMap(data)
	if err != nil {
		return nil, err
	}
	session.RefreshMetadata()
	return session, nil
}

// Save persists session state.
func (s *ChatSessionStore) Save(session *ChatSession) error {
	session.RefreshMetadata()
	session.UpdatedAt = time.Now().UTC()
	if err := s.store.Save(s.key(session.ID), session.toMap()); err != nil {
		return err
	}
	return s.upsertIndexEntry(session)
}

// ListIndexedSessions returns compact session metadata from the manifest.
func (s *ChatSessionStore) ListIndexedSessions() ([]ChatSessionIndexEntry, error) {
	idx, err := s.loadIndex()
	if err != nil {
		return nil, err
	}
	entries := append([]ChatSessionIndexEntry(nil), idx.Sessions...)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].UpdatedAt.After(entries[j].UpdatedAt)
	})
	return entries, nil
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

// RefreshMetadata derives searchable fields from persisted chat content.
func (c *ChatSession) RefreshMetadata() {
	if strings.TrimSpace(c.Title) == "" || c.Title == c.ID {
		c.Title = deriveSessionTitle(c)
	}
	c.ToolNames = uniqueSortedStrings(append(c.ToolNames, deriveToolNames(c)...))
	c.Tags = uniqueSortedStrings(append(c.Tags, deriveSessionTags(c)...))
	if strings.TrimSpace(c.Summary) == "" {
		c.Summary = deriveSessionSummary(c)
	}
	if c.Metadata == nil {
		c.Metadata = map[string]any{}
	}
	c.Metadata["message_count"] = len(c.Messages)
	c.Metadata["step_count"] = len(c.StepRecords)
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
		"id": c.ID, "title": c.Title, "tags": c.Tags, "summary": c.Summary,
		"tool_names": c.ToolNames, "metadata": c.Metadata, "status": c.Status,
		"created_at": c.CreatedAt.Format(time.RFC3339),
		"updated_at": c.UpdatedAt.Format(time.RFC3339),
		"messages":   msgs, "step_records": records, "step_counter": c.StepCounter,
	}
}

func chatSessionFromMap(data map[string]any) (*ChatSession, error) {
	session := &ChatSession{
		ID: stringField(data, "id"), Title: stringField(data, "title"), Summary: stringField(data, "summary"),
		Status:      stringField(data, "status"),
		StepCounter: intField(data, "step_counter"),
	}
	session.Tags = stringSliceField(data, "tags")
	session.ToolNames = stringSliceField(data, "tool_names")
	if metadata, ok := data["metadata"].(map[string]any); ok {
		session.Metadata = metadata
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

func stringSliceField(m map[string]any, k string) []string {
	raw, ok := m[k].([]any)
	if !ok {
		return nil
	}
	items := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			items = append(items, s)
		}
	}
	return uniqueSortedStrings(items)
}

func (s *ChatSessionStore) loadIndex() (*ChatSessionIndex, error) {
	data, err := s.store.Load(s.indexKey())
	if err != nil {
		if strings.Contains(err.Error(), "corrupt state file") {
			return s.rebuildIndex()
		}
		return nil, err
	}
	if data == nil {
		return &ChatSessionIndex{Version: 1}, nil
	}
	return chatSessionIndexFromMap(data), nil
}

func (s *ChatSessionStore) rebuildIndex() (*ChatSessionIndex, error) {
	ids, err := s.ListSessionIDs()
	if err != nil {
		return nil, err
	}
	idx := &ChatSessionIndex{Version: 1}
	for _, id := range ids {
		session, err := s.Load(id)
		if err != nil {
			return nil, err
		}
		if session == nil {
			continue
		}
		idx.Sessions = append(idx.Sessions, indexEntryFromSession(session))
	}
	sort.Slice(idx.Sessions, func(i, j int) bool {
		return idx.Sessions[i].UpdatedAt.After(idx.Sessions[j].UpdatedAt)
	})
	if err := s.saveIndex(idx); err != nil {
		return nil, err
	}
	return idx, nil
}

func (s *ChatSessionStore) saveIndex(index *ChatSessionIndex) error {
	index.Version = 1
	index.UpdatedAt = time.Now().UTC()
	return s.store.Save(s.indexKey(), index.toMap())
}

func (s *ChatSessionStore) upsertIndexEntry(session *ChatSession) error {
	idx, err := s.loadIndex()
	if err != nil {
		return err
	}
	entry := indexEntryFromSession(session)
	replaced := false
	for i := range idx.Sessions {
		if idx.Sessions[i].ID == session.ID {
			idx.Sessions[i] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		idx.Sessions = append(idx.Sessions, entry)
	}
	sort.Slice(idx.Sessions, func(i, j int) bool {
		return idx.Sessions[i].UpdatedAt.After(idx.Sessions[j].UpdatedAt)
	})
	return s.saveIndex(idx)
}

func indexEntryFromSession(session *ChatSession) ChatSessionIndexEntry {
	return ChatSessionIndexEntry{
		ID: session.ID, Title: session.Title, Tags: append([]string(nil), session.Tags...),
		Summary: session.Summary, ToolNames: append([]string(nil), session.ToolNames...), Status: session.Status,
		CreatedAt: session.CreatedAt, UpdatedAt: session.UpdatedAt, MessageCount: len(session.Messages),
		StepCount: len(session.StepRecords), Metadata: cloneMetadata(session.Metadata),
	}
}

func chatSessionIndexFromMap(data map[string]any) *ChatSessionIndex {
	idx := &ChatSessionIndex{Version: intField(data, "version")}
	if idx.Version == 0 {
		idx.Version = 1
	}
	if t, err := time.Parse(time.RFC3339, stringField(data, "updated_at")); err == nil {
		idx.UpdatedAt = t
	}
	if rawSessions, ok := data["sessions"].([]any); ok {
		for _, item := range rawSessions {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			entry := ChatSessionIndexEntry{
				ID: stringField(m, "id"), Title: stringField(m, "title"), Tags: stringSliceField(m, "tags"),
				Summary: stringField(m, "summary"), ToolNames: stringSliceField(m, "tool_names"),
				Status: stringField(m, "status"), MessageCount: intField(m, "message_count"), StepCount: intField(m, "step_count"),
			}
			if t, err := time.Parse(time.RFC3339, stringField(m, "created_at")); err == nil {
				entry.CreatedAt = t
			}
			if t, err := time.Parse(time.RFC3339, stringField(m, "updated_at")); err == nil {
				entry.UpdatedAt = t
			}
			if metadata, ok := m["metadata"].(map[string]any); ok {
				entry.Metadata = metadata
			}
			if entry.ID != "" {
				idx.Sessions = append(idx.Sessions, entry)
			}
		}
	}
	return idx
}

func (idx *ChatSessionIndex) toMap() map[string]any {
	sessions := make([]map[string]any, 0, len(idx.Sessions))
	for _, entry := range idx.Sessions {
		sessions = append(sessions, map[string]any{
			"id": entry.ID, "title": entry.Title, "tags": entry.Tags, "summary": entry.Summary,
			"tool_names": entry.ToolNames, "status": entry.Status,
			"created_at": entry.CreatedAt.Format(time.RFC3339), "updated_at": entry.UpdatedAt.Format(time.RFC3339),
			"message_count": entry.MessageCount, "step_count": entry.StepCount, "metadata": entry.Metadata,
		})
	}
	return map[string]any{"version": idx.Version, "updated_at": idx.UpdatedAt.Format(time.RFC3339), "sessions": sessions}
}

func deriveSessionTitle(session *ChatSession) string {
	for _, msg := range session.Messages {
		if msg.Role == llm.RoleUser && strings.TrimSpace(msg.Content) != "" {
			return truncateRunes(strings.TrimSpace(msg.Content), 80)
		}
	}
	return session.ID
}

func deriveSessionSummary(session *ChatSession) string {
	for i := len(session.StepRecords) - 1; i >= 0; i-- {
		if strings.TrimSpace(session.StepRecords[i].Summary) != "" {
			return truncateRunes(strings.TrimSpace(session.StepRecords[i].Summary), 240)
		}
	}
	for i := len(session.Messages) - 1; i >= 0; i-- {
		msg := session.Messages[i]
		if msg.Role != llm.RoleSystem && strings.TrimSpace(msg.Content) != "" {
			return truncateRunes(strings.TrimSpace(msg.Content), 240)
		}
	}
	return ""
}

func deriveToolNames(session *ChatSession) []string {
	var names []string
	for _, msg := range session.Messages {
		for _, call := range msg.ToolCalls {
			if strings.TrimSpace(call.Name) != "" {
				names = append(names, call.Name)
			}
		}
	}
	for _, rec := range session.StepRecords {
		if strings.TrimSpace(rec.ToolName) != "" {
			names = append(names, rec.ToolName)
		}
	}
	return names
}

func deriveSessionTags(session *ChatSession) []string {
	var tags []string
	for _, msg := range session.Messages {
		for _, call := range msg.ToolCalls {
			for _, key := range []string{"code", "symbol", "ticker"} {
				if v := strings.TrimSpace(strFromMap(call.Arguments, key)); v != "" {
					tags = append(tags, strings.ToUpper(v))
				}
			}
		}
	}
	return tags
}

func strFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if s, ok := m[key].(string); ok {
		return s
	}
	return ""
}

func uniqueSortedStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func cloneMetadata(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
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
