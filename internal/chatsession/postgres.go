package chatsession

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// PostgresSessionStore persists chat sessions in PostgreSQL.
type PostgresSessionStore struct {
	db *sql.DB
}

// NewPostgresSessionStore creates a PostgreSQL-backed session store.
func NewPostgresSessionStore(db *sql.DB) *PostgresSessionStore {
	return &PostgresSessionStore{db: db}
}

// Create allocates and saves a new active session with the system prompt.
func (s *PostgresSessionStore) Create() (*ChatSession, error) {
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
func (s *PostgresSessionStore) Load(sessionID string) (*ChatSession, error) {
	ctx := context.Background()
	var (
		title, status, summary, userID string
		created, updated                 time.Time
		stepCounter                      int
		tagsJSON, toolNamesJSON          []byte
		metadataJSON, messagesJSON       []byte
		stepRecordsJSON                  []byte
	)
	err := s.db.QueryRowContext(ctx, `
        SELECT user_id, title, status, summary, created_at, updated_at, step_counter,
               tags_json, tool_names_json, metadata_json, messages_json, step_records_json
        FROM chat_sessions WHERE id=$1`, sessionID,
	).Scan(&userID, &title, &status, &summary, &created, &updated, &stepCounter,
		&tagsJSON, &toolNamesJSON, &metadataJSON, &messagesJSON, &stepRecordsJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load session %s: %w", sessionID, err)
	}
	session := &ChatSession{
		ID: sessionID, Title: title, Status: status, Summary: summary,
		StepCounter: stepCounter, CreatedAt: created, UpdatedAt: updated,
		Tags:        decodeStringSlice(string(tagsJSON)),
		ToolNames:   decodeStringSlice(string(toolNamesJSON)),
		Metadata:    decodeMap(string(metadataJSON)),
		Messages:    decodeMessages(string(messagesJSON)),
		StepRecords: decodeStepRecords(string(stepRecordsJSON)),
	}
	if session.Metadata == nil {
		session.Metadata = map[string]any{}
	}
	if userID != "" {
		session.Metadata["user_id"] = userID
	}
	session.RefreshMetadata()
	return session, nil
}

// Save persists session state, upserting by id.
func (s *PostgresSessionStore) Save(session *ChatSession) error {
	session.RefreshMetadata()
	session.UpdatedAt = time.Now().UTC()

	tagsJSON, _ := json.Marshal(session.Tags)
	toolNamesJSON, _ := json.Marshal(session.ToolNames)
	metadataJSON, _ := json.Marshal(session.Metadata)
	messagesJSON, _ := json.Marshal(session.Messages)
	stepRecordsJSON, _ := json.Marshal(session.StepRecords)
	userID := userIDFromMetadata(session.Metadata)

	ctx := context.Background()
	_, err := s.db.ExecContext(ctx, `
        INSERT INTO chat_sessions
            (id, user_id, title, status, created_at, updated_at, step_counter,
             tags_json, summary, tool_names_json, metadata_json,
             messages_json, step_records_json)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9,$10::jsonb,$11::jsonb,$12::jsonb,$13::jsonb)
        ON CONFLICT (id) DO UPDATE SET
            user_id=EXCLUDED.user_id, title=EXCLUDED.title, status=EXCLUDED.status,
            updated_at=EXCLUDED.updated_at, step_counter=EXCLUDED.step_counter,
            tags_json=EXCLUDED.tags_json, summary=EXCLUDED.summary,
            tool_names_json=EXCLUDED.tool_names_json, metadata_json=EXCLUDED.metadata_json,
            messages_json=EXCLUDED.messages_json, step_records_json=EXCLUDED.step_records_json`,
		session.ID, userID, session.Title, session.Status,
		session.CreatedAt, session.UpdatedAt, session.StepCounter,
		string(tagsJSON), session.Summary, string(toolNamesJSON),
		string(metadataJSON), string(messagesJSON), string(stepRecordsJSON),
	)
	if err != nil {
		return fmt.Errorf("save session %s: %w", session.ID, err)
	}
	return nil
}

// ListIndexedSessions returns compact session metadata ordered by updated_at desc.
func (s *PostgresSessionStore) ListIndexedSessions() ([]ChatSessionIndexEntry, error) {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, title, tags_json, summary, tool_names_json, status,
               created_at, updated_at, metadata_json,
               COALESCE(jsonb_array_length(messages_json), 0),
               COALESCE(jsonb_array_length(step_records_json), 0)
        FROM chat_sessions ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChatSessionIndexEntry
	for rows.Next() {
		var (
			id, title, status, summary string
			tagsJSON, toolNamesJSON    []byte
			metadataJSON               []byte
			created, updated           time.Time
			msgCount, stepCount        int
		)
		if err := rows.Scan(&id, &title, &tagsJSON, &summary, &toolNamesJSON, &status,
			&created, &updated, &metadataJSON, &msgCount, &stepCount); err != nil {
			return nil, err
		}
		out = append(out, ChatSessionIndexEntry{
			ID: id, Title: title, Tags: decodeStringSlice(string(tagsJSON)), Summary: summary,
			ToolNames: decodeStringSlice(string(toolNamesJSON)), Status: status,
			MessageCount: msgCount, StepCount: stepCount,
			Metadata: decodeMap(string(metadataJSON)),
			CreatedAt: created, UpdatedAt: updated,
		})
	}
	return out, rows.Err()
}

// ListSessionIDs returns all chat session ids ordered by updated_at desc.
func (s *PostgresSessionStore) ListSessionIDs() ([]string, error) {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM chat_sessions ORDER BY updated_at DESC`)
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

// CopyAllSessions migrates every session from src into this store.
func CopyAllSessions(src SessionStore, dest *PostgresSessionStore) (migrated, skipped int, err error) {
	ids, err := src.ListSessionIDs()
	if err != nil {
		return 0, 0, err
	}
	for _, id := range ids {
		s, loadErr := src.Load(id)
		if loadErr != nil || s == nil {
			skipped++
			continue
		}
		if saveErr := dest.Save(s); saveErr != nil {
			skipped++
			continue
		}
		migrated++
	}
	return migrated, skipped, nil
}

func userIDFromMetadata(m map[string]any) string {
	if m == nil {
		return ""
	}
	if v, ok := m["user_id"].(string); ok {
		return v
	}
	return ""
}
