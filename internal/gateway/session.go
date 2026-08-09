package gateway

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// SessionMap persists gateway session key → chatsession ID.
type SessionMap struct {
	mu   sync.Mutex
	path string
	Data map[string]string `json:"sessions"`
}

// NewSessionMap loads or creates a map under workspaceDir/gateway/sessions.json.
func NewSessionMap(workspaceDir string) (*SessionMap, error) {
	dir := filepath.Join(workspaceDir, "gateway")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "sessions.json")
	m := &SessionMap{path: path, Data: map[string]string{}}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return m, nil
		}
		return nil, err
	}
	if len(raw) == 0 {
		return m, nil
	}
	var file struct {
		Sessions map[string]string `json:"sessions"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("gateway sessions: %w", err)
	}
	if file.Sessions != nil {
		m.Data = file.Sessions
	}
	return m, nil
}

// Get returns the chat session id for key, if any.
func (m *SessionMap) Get(key string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.Data[key]
	return id, ok
}

// Put stores and persists the mapping.
func (m *SessionMap) Put(key, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Data[key] = sessionID
	return m.saveLocked()
}

func (m *SessionMap) saveLocked() error {
	raw, err := json.MarshalIndent(struct {
		Sessions map[string]string `json:"sessions"`
	}{Sessions: m.Data}, "", "  ")
	if err != nil {
		return err
	}
	tmp := m.path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, m.path)
}
