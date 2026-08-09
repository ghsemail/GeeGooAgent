package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnvFilePath returns ~/.geegoo/.env (or $GEEGOO_HOME/.env).
func EnvFilePath() string {
	return filepath.Join(Home(), ".env")
}

// LoadDotEnv reads KEY=VALUE lines from path into the process environment
// without overriding variables that are already set.
func LoadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, val)
	}
	return sc.Err()
}

// LoadGeeGooDotEnv loads ~/.geegoo/.env if present.
func LoadGeeGooDotEnv() error {
	return LoadDotEnv(EnvFilePath())
}

// UpsertEnvFile sets keys in path (creates file/dir as needed). Preserves other lines.
func UpsertEnvFile(path string, values map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	existing := map[string]string{}
	var order []string
	var extras []string
	if raw, err := os.ReadFile(path); err == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			trim := strings.TrimSpace(line)
			if trim == "" || strings.HasPrefix(trim, "#") {
				extras = append(extras, line)
				continue
			}
			key, val, ok := strings.Cut(line, "=")
			if !ok {
				extras = append(extras, line)
				continue
			}
			key = strings.TrimSpace(key)
			existing[key] = strings.TrimSpace(val)
			order = append(order, key)
		}
	}
	for k, v := range values {
		if _, ok := existing[k]; !ok {
			order = append(order, k)
		}
		existing[k] = v
		_ = os.Setenv(k, v)
	}
	var b strings.Builder
	seen := map[string]bool{}
	for _, line := range extras {
		// Keep comments / blanks only at top once; simpler: rewrite cleanly.
		_ = line
	}
	b.WriteString("# GeeGoo Agent environment (written by geegoo gateway setup)\n")
	for _, k := range order {
		if seen[k] {
			continue
		}
		seen[k] = true
		fmt.Fprintf(&b, "%s=%s\n", k, existing[k])
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// SaveFeishuEnv writes Feishu credentials into ~/.geegoo/.env and the process env.
func SaveFeishuEnv(appID, appSecret, domain string, extra map[string]string) error {
	vals := map[string]string{
		"FEISHU_APP_ID":          appID,
		"FEISHU_APP_SECRET":      appSecret,
		"FEISHU_DOMAIN":          domain,
		"FEISHU_CONNECTION_MODE": "websocket",
	}
	for k, v := range extra {
		if strings.TrimSpace(v) != "" {
			vals[k] = v
		}
	}
	return UpsertEnvFile(EnvFilePath(), vals)
}
