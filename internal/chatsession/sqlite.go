package chatsession

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// SQLiteSessionStore persists chat sessions in SQLite (WAL + FTS5).
// It implements SessionStore.
type SQLiteSessionStore struct {
	db *infra.DB
}

// NewSQLiteSessionStore creates a SQLite-backed session store.
func NewSQLiteSessionStore(db *infra.DB) *SQLiteSessionStore {
	return &SQLiteSessionStore{db: db}
}

// Create allocates and saves a new active session with the system prompt.
func (s *SQLiteSessionStore) Create() (*ChatSession, error) {
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
func (s *SQLiteSessionStore) Load(sessionID string) (*ChatSession, error) {
	ctx := context.Background()
	var (
		title, status, summary           string
		created, updated                 string
		tagsJSON, toolNamesJSON          string
		metadataJSON, messagesJSON       string
		stepRecordsJSON                  string
		stepCounter                      int
	)
	err := s.db.SQL().QueryRowContext(ctx, `
        SELECT title, status, summary, created_at, updated_at, step_counter,
               tags_json, tool_names_json, metadata_json, messages_json, step_records_json
        FROM chat_sessions WHERE id=?`, sessionID,
	).Scan(&title, &status, &summary, &created, &updated, &stepCounter,
		&tagsJSON, &toolNamesJSON, &metadataJSON, &messagesJSON, &stepRecordsJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load session %s: %w", sessionID, err)
	}
	session := &ChatSession{
		ID:          sessionID,
		Title:       title,
		Status:      status,
		Summary:     summary,
		StepCounter: stepCounter,
		Tags:        decodeStringSlice(tagsJSON),
		ToolNames:   decodeStringSlice(toolNamesJSON),
		Metadata:    decodeMap(metadataJSON),
	}
	session.Messages = decodeMessages(messagesJSON)
	session.StepRecords = decodeStepRecords(stepRecordsJSON)
	if t, err := time.Parse(time.RFC3339Nano, created); err == nil {
		session.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339Nano, updated); err == nil {
		session.UpdatedAt = t
	}
	session.RefreshMetadata()
	return session, nil
}

// Save persists session state, upserting by id.
func (s *SQLiteSessionStore) Save(session *ChatSession) error {
	session.RefreshMetadata()
	session.UpdatedAt = time.Now().UTC()

	tagsJSON, _ := json.Marshal(session.Tags)
	toolNamesJSON, _ := json.Marshal(session.ToolNames)
	metadataJSON, _ := json.Marshal(session.Metadata)
	messagesJSON, _ := json.Marshal(session.Messages)
	stepRecordsJSON, _ := json.Marshal(session.StepRecords)

	ctx := context.Background()
	_, err := s.db.SQL().ExecContext(ctx, `
        INSERT INTO chat_sessions
            (id, title, status, created_at, updated_at, step_counter,
             tags_json, summary, tool_names_json, metadata_json,
             messages_json, step_records_json)
        VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
        ON CONFLICT(id) DO UPDATE SET
            title=excluded.title, status=excluded.status, updated_at=excluded.updated_at,
            step_counter=excluded.step_counter, tags_json=excluded.tags_json,
            summary=excluded.summary, tool_names_json=excluded.tool_names_json,
            metadata_json=excluded.metadata_json, messages_json=excluded.messages_json,
            step_records_json=excluded.step_records_json`,
		session.ID, session.Title, session.Status,
		session.CreatedAt.Format(time.RFC3339Nano), session.UpdatedAt.Format(time.RFC3339Nano),
		session.StepCounter, string(tagsJSON), session.Summary, string(toolNamesJSON),
		string(metadataJSON), string(messagesJSON), string(stepRecordsJSON),
	)
	if err != nil {
		return fmt.Errorf("save session %s: %w", session.ID, err)
	}
	return s.upsertFTS(ctx, session)
}

func (s *SQLiteSessionStore) upsertFTS(ctx context.Context, session *ChatSession) error {
	// Replace any existing FTS rows for this session, then insert a fresh one.
	_, _ = s.db.SQL().ExecContext(ctx,
		"DELETE FROM chat_sessions_fts WHERE session_id=?", session.ID)
	_, err := s.db.SQL().ExecContext(ctx,
		"INSERT INTO chat_sessions_fts(session_id, title, summary) VALUES(?, ?, ?)",
		session.ID, session.Title, session.Summary)
	return err
}

// ListIndexedSessions returns compact session metadata ordered by updated_at desc.
func (s *SQLiteSessionStore) ListIndexedSessions() ([]ChatSessionIndexEntry, error) {
	ctx := context.Background()
	rows, err := s.db.SQL().QueryContext(ctx, `
        SELECT id, title, tags_json, summary, tool_names_json, status,
               created_at, updated_at, metadata_json,
               length(messages_json), length(step_records_json)
        FROM chat_sessions ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChatSessionIndexEntry
	for rows.Next() {
		var (
			id, title, status, summary       string
			tagsJSON, toolNamesJSON          string
			created, updated                 string
			metadataJSON                     string
			msgLen, recLen                   int
		)
		if err := rows.Scan(&id, &title, &tagsJSON, &summary, &toolNamesJSON, &status,
			&created, &updated, &metadataJSON, &msgLen, &recLen); err != nil {
			return nil, err
		}
		entry := ChatSessionIndexEntry{
			ID: id, Title: title, Tags: decodeStringSlice(tagsJSON), Summary: summary,
			ToolNames: decodeStringSlice(toolNamesJSON), Status: status,
			MessageCount: msgLen, StepCount: recLen, Metadata: decodeMap(metadataJSON),
		}
		if t, err := time.Parse(time.RFC3339Nano, created); err == nil {
			entry.CreatedAt = t
		}
		if t, err := time.Parse(time.RFC3339Nano, updated); err == nil {
			entry.UpdatedAt = t
		}
		out = append(out, entry)
	}
	return out, rows.Err()
}

// ListSessionIDs returns all chat session ids ordered by updated_at desc.
func (s *SQLiteSessionStore) ListSessionIDs() ([]string, error) {
	ctx := context.Background()
	rows, err := s.db.SQL().QueryContext(ctx,
		"SELECT id FROM chat_sessions ORDER BY updated_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// SearchFTS uses the FTS5 index for keyword search; returns ranked session ids.
func (s *SQLiteSessionStore) SearchFTS(query string, limit int) ([]string, error) {
	q := normalizeFTSQuery(query)
	if q == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}
	ctx := context.Background()
	rows, err := s.db.SQL().QueryContext(ctx, `
        SELECT session_id FROM chat_sessions_fts
        WHERE chat_sessions_fts MATCH ?
        ORDER BY rank LIMIT ?`, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func decodeStringSlice(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	if json.Unmarshal([]byte(raw), &out) != nil {
		return nil
	}
	return uniqueSortedStrings(out)
}

func decodeMap(raw string) map[string]any {
	if raw == "" {
		return nil
	}
	var out map[string]any
	if json.Unmarshal([]byte(raw), &out) != nil {
		return nil
	}
	return out
}

func decodeMessages(raw string) []llm.Message {
	if raw == "" || raw == "null" {
		return nil
	}
	var out []llm.Message
	if json.Unmarshal([]byte(raw), &out) != nil {
		return nil
	}
	return out
}

func decodeStepRecords(raw string) []ChatStepRecord {
	if raw == "" || raw == "null" {
		return nil
	}
	var out []ChatStepRecord
	if json.Unmarshal([]byte(raw), &out) != nil {
		return nil
	}
	return out
}

// normalizeFTSQuery turns a free-form user query into an FTS5 MATCH expression
// by quoting each whitespace-delimited token.
func normalizeFTSQuery(query string) string {
	tokens := []string{}
	for _, t := range splitFields(query) {
		if t == "" {
			continue
		}
		tokens = append(tokens, "\""+escapeFTS(t)+"\"")
	}
	sort.Strings(tokens)
	return joinSpaces(tokens)
}

func escapeFTS(s string) string {
	s = replaceAll(s, "\"", "")
	return s
}

func splitFields(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func joinSpaces(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " "
		}
		out += p
	}
	return out
}

func replaceAll(s, old, new string) string {
	out := ""
	for {
		i := indexOf(s, old)
		if i < 0 {
			out += s
			break
		}
		out += s[:i] + new
		s = s[i+len(old):]
	}
	return out
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
