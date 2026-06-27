package infra

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StateStore persists JSON documents under a root directory.
type StateStore struct {
	root string
}

// NewStateStore creates a file-backed store.
func NewStateStore(root string) *StateStore {
	return &StateStore{root: root}
}

func (s *StateStore) pathFor(key string) (string, error) {
	if strings.Contains(key, "..") || strings.HasPrefix(key, "/") {
		return "", fmt.Errorf("invalid state key: %q", key)
	}
	return filepath.Join(s.root, key+".json"), nil
}

// Save writes JSON for key.
func (s *StateStore) Save(key string, data map[string]any) error {
	path, err := s.pathFor(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// Load reads JSON for key.
func (s *StateStore) Load(key string) (map[string]any, error) {
	path, err := s.pathFor(key)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("corrupt state file %s: %w", path, err)
	}
	return data, nil
}

// Checkpoint records workflow progress.
type Checkpoint struct {
	SessionID string         `json:"session_id"`
	Step      int            `json:"step"`
	Skill     string         `json:"skill"`
	Status    string         `json:"status"`
	Working   map[string]any `json:"working"`
	LastTool  string         `json:"last_tool"`
}

// CheckpointManager saves latest checkpoint per session.
type CheckpointManager struct {
	store *StateStore
}

// NewCheckpointManager creates a checkpoint manager.
func NewCheckpointManager(store *StateStore) *CheckpointManager {
	return &CheckpointManager{store: store}
}

func (m *CheckpointManager) key(sessionID string) string {
	return "checkpoints/" + sessionID + "/latest"
}

// Save persists a checkpoint.
func (m *CheckpointManager) Save(cp Checkpoint) error {
	return m.store.Save(m.key(cp.SessionID), map[string]any{
		"session_id": cp.SessionID,
		"step":       cp.Step,
		"skill":      cp.Skill,
		"status":     cp.Status,
		"working":    cp.Working,
		"last_tool":  cp.LastTool,
	})
}

// LoadLatest returns the latest checkpoint for a session.
func (m *CheckpointManager) LoadLatest(sessionID string) (*Checkpoint, error) {
	data, err := m.store.Load(m.key(sessionID))
	if err != nil || data == nil {
		return nil, err
	}
	cp := &Checkpoint{
		SessionID: stringField(data, "session_id"),
		Step:      intField(data, "step"),
		Skill:     stringField(data, "skill"),
		Status:    stringField(data, "status"),
		LastTool:  stringField(data, "last_tool"),
	}
	if w, ok := data["working"].(map[string]any); ok {
		cp.Working = w
	}
	return cp, nil
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
	default:
		return 0
	}
}

// WorkspaceGuard restricts file paths to workspace root.
type WorkspaceGuard struct {
	root string
}

// NewWorkspaceGuard creates a guard for workspaceRoot.
func NewWorkspaceGuard(root string) (*WorkspaceGuard, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &WorkspaceGuard{root: abs}, nil
}

// Resolve joins relative path under workspace.
func (g *WorkspaceGuard) Resolve(rel string) (string, error) {
	if filepath.IsAbs(rel) || strings.Contains(rel, "..") {
		return "", fmt.Errorf("absolute paths not allowed: %s", rel)
	}
	target, err := filepath.Abs(filepath.Join(g.root, rel))
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(target, g.root+string(os.PathSeparator)) && target != g.root {
		return "", fmt.Errorf("path outside workspace: %s", rel)
	}
	return target, nil
}
