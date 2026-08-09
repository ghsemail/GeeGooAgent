package gateway

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// HeartbeatFile is written by `geegoo gateway run` so dashboard status can see liveness.
func HeartbeatFile() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = "."
	}
	return filepath.Join(home, ".geegoo", "geegoo-agent", "gateway.heartbeat.json")
}

// HeartbeatSnapshot is persisted by the long-running gateway process.
type HeartbeatSnapshot struct {
	Platform   string                        `json:"platform"`
	Connected  bool                          `json:"connected"`
	Configured bool                          `json:"configured"`
	Detail     string                        `json:"detail"`
	PID        int                           `json:"pid"`
	UpdatedAt  time.Time                     `json:"updated_at"`
	Users      map[string]UserHeartbeatStatus `json:"users,omitempty"`
}

// UserHeartbeatStatus is one tenant's Feishu adapter liveness.
type UserHeartbeatStatus struct {
	Connected  bool   `json:"connected"`
	Configured bool   `json:"configured"`
	BotName    string `json:"bot_name,omitempty"`
	AppIDMask  string `json:"app_id_masked,omitempty"`
	Detail     string `json:"detail,omitempty"`
}

// WriteHeartbeat atomically updates the heartbeat file.
func WriteHeartbeat(snap HeartbeatSnapshot) error {
	path := HeartbeatFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if snap.UpdatedAt.IsZero() {
		snap.UpdatedAt = time.Now().UTC()
	}
	if snap.PID == 0 {
		snap.PID = os.Getpid()
	}
	raw, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ReadHeartbeat loads the last heartbeat; ok=false if missing/unreadable.
func ReadHeartbeat() (HeartbeatSnapshot, bool) {
	raw, err := os.ReadFile(HeartbeatFile())
	if err != nil {
		return HeartbeatSnapshot{}, false
	}
	var snap HeartbeatSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return HeartbeatSnapshot{}, false
	}
	return snap, true
}

// HeartbeatFresh reports whether the gateway process reported recently.
func HeartbeatFresh(snap HeartbeatSnapshot, maxAge time.Duration) bool {
	if snap.UpdatedAt.IsZero() {
		return false
	}
	return time.Since(snap.UpdatedAt.UTC()) <= maxAge
}
