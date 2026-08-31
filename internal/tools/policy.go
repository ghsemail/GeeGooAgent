package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	policyMu    sync.RWMutex
	policyRules []policyRule
)

type policyFile struct {
	Rules []policyRule `json:"rules"`
}

type policyRule struct {
	Match  string       `json:"match"`
	Action PolicyAction `json:"action"`
}

// LoadPolicyFile reads declarative tool policy rules (Codex execpolicy style).
// Supports JSON; also accepts .yaml extension with a minimal line parser.
// Missing file is not an error.
func LoadPolicyFile(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var file policyFile
	if strings.HasSuffix(strings.ToLower(path), ".yaml") || strings.HasSuffix(strings.ToLower(path), ".yml") {
		file.Rules = parseSimpleYAMLRules(string(raw))
	} else if err := json.Unmarshal(raw, &file); err != nil {
		return err
	}
	policyMu.Lock()
	policyRules = file.Rules
	policyMu.Unlock()
	return nil
}

// parseSimpleYAMLRules handles minimal rules blocks without a YAML dependency.
func parseSimpleYAMLRules(raw string) []policyRule {
	var rules []policyRule
	var cur *policyRule
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == "rules:" {
			continue
		}
		if strings.HasPrefix(line, "- match:") {
			if cur != nil {
				rules = append(rules, *cur)
			}
			cur = &policyRule{Match: trimYAMLValue(strings.TrimPrefix(line, "- match:"))}
			continue
		}
		if strings.HasPrefix(line, "match:") && cur == nil {
			cur = &policyRule{Match: trimYAMLValue(strings.TrimPrefix(line, "match:"))}
			continue
		}
		if strings.HasPrefix(line, "action:") && cur != nil {
			cur.Action = PolicyAction(trimYAMLValue(strings.TrimPrefix(line, "action:")))
		}
	}
	if cur != nil {
		rules = append(rules, *cur)
	}
	return rules
}

func trimYAMLValue(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'`)
	return s
}

// DefaultPolicyPath returns ~/.geegoo/tool_policy.yaml when GEEGOO_HOME is set.
func DefaultPolicyPath(home string) string {
	home = strings.TrimSpace(home)
	if home == "" {
		return ""
	}
	jsonPath := filepath.Join(home, "tool_policy.json")
	if _, err := os.Stat(jsonPath); err == nil {
		return jsonPath
	}
	return filepath.Join(home, "tool_policy.yaml")
}

func matchPolicyRule(name string) PolicyAction {
	policyMu.RLock()
	rules := policyRules
	policyMu.RUnlock()
	for _, rule := range rules {
		if rule.Match == "" || rule.Action == "" {
			continue
		}
		if matchGlob(name, rule.Match) {
			return rule.Action
		}
	}
	return ""
}

// matchGlob supports simple patterns: exact name, * suffix/prefix, a|b alternation.
func matchGlob(name, pattern string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	if strings.Contains(pattern, "|") {
		for _, part := range strings.Split(pattern, "|") {
			if matchGlob(name, strings.TrimSpace(part)) {
				return true
			}
		}
		return false
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(name, strings.TrimSuffix(pattern, "*"))
	}
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(name, strings.TrimPrefix(pattern, "*"))
	}
	return name == pattern
}

// PolicyPrompt is the action requiring user confirmation (plan gate).
const PolicyPrompt PolicyAction = "prompt"

// PolicyAllow permits execution without confirmation.
const PolicyAllow PolicyAction = "allow"
