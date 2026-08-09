package feishustore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var safeUserID = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// Creds is one ops-console user's Feishu bot binding.
type Creds struct {
	UserID       string   `json:"user_id"`
	MCPToken     string   `json:"mcp_token,omitempty"`
	AppID        string   `json:"app_id"`
	AppSecret    string   `json:"app_secret"`
	Domain       string   `json:"domain,omitempty"`
	BotName      string   `json:"bot_name,omitempty"`
	BotOpenID    string   `json:"bot_open_id,omitempty"`
	AllowedUsers []string `json:"allowed_users,omitempty"`
	HomeChannel  string   `json:"home_channel,omitempty"`
	GroupPolicy  string   `json:"group_policy,omitempty"`
	Enabled      bool     `json:"enabled"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
}

func SanitizeUserID(userID string) string {
	safe := safeUserID.ReplaceAllString(strings.TrimSpace(userID), "_")
	if safe == "" {
		return "anonymous"
	}
	return safe
}

func Dir(outputDir string) string {
	if strings.TrimSpace(outputDir) == "" {
		outputDir = "."
	}
	return filepath.Join(outputDir, "user_gateway_feishu")
}

func Path(outputDir, userID string) string {
	return filepath.Join(Dir(outputDir), SanitizeUserID(userID)+".json")
}

func ReloadMarker(outputDir string) string {
	return filepath.Join(Dir(outputDir), ".reload")
}

func TouchReload(outputDir string) error {
	dir := Dir(outputDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(ReloadMarker(outputDir), []byte(time.Now().UTC().Format(time.RFC3339Nano)+"\n"), 0o644)
}

func ReloadToken(outputDir string) (string, error) {
	raw, err := os.ReadFile(ReloadMarker(outputDir))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}

func Load(outputDir, userID string) (*Creds, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(Path(outputDir, userID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var c Creds
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	if c.UserID == "" {
		c.UserID = userID
	}
	return &c, nil
}

func Save(outputDir string, c *Creds) error {
	if c == nil || strings.TrimSpace(c.UserID) == "" {
		return os.ErrInvalid
	}
	c.UserID = strings.TrimSpace(c.UserID)
	if c.Domain == "" {
		c.Domain = "feishu"
	}
	if c.GroupPolicy == "" {
		c.GroupPolicy = "allowlist"
	}
	c.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	dir := Dir(outputDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	path := Path(outputDir, c.UserID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return TouchReload(outputDir)
}

func List(outputDir string) ([]Creds, error) {
	dir := Dir(outputDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]Creds, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var c Creds
		if err := json.Unmarshal(raw, &c); err != nil {
			continue
		}
		if strings.TrimSpace(c.AppID) == "" || strings.TrimSpace(c.AppSecret) == "" {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

// Fingerprint detects credential changes for hot reload.
func (c Creds) Fingerprint() string {
	return strings.Join([]string{
		c.UserID, c.AppID, c.AppSecret, c.Domain, c.MCPToken,
		strings.Join(c.AllowedUsers, ","),
		boolStr(c.Enabled),
	}, "|")
}

func boolStr(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

// Configured reports whether credentials are present.
func (c *Creds) Configured() bool {
	return c != nil && strings.TrimSpace(c.AppID) != "" && strings.TrimSpace(c.AppSecret) != ""
}
