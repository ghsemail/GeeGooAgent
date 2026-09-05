package chatsession

import (
	"encoding/json"
	"strings"
)

const (
	metaLastTurnPlan       = "last_turn_plan"
	metaLastTurnToolsCalls = "last_turn_tools_called"
)

// TurnPlanSnapshot is stored on ChatSession.Metadata after each agent turn.
type TurnPlanSnapshot struct {
	Domain     string   `json:"domain"`
	Mode       string   `json:"mode"`
	SOP        bool     `json:"sop"`
	ToolsAllow []string `json:"tools_allow,omitempty"`
}

// SyncLastTurnPlan writes the routing snapshot for eval verification.
func (c *ChatSession) SyncLastTurnPlan(domain, mode string, sop bool, toolsAllow []string) {
	if c == nil {
		return
	}
	if c.Metadata == nil {
		c.Metadata = map[string]any{}
	}
	snap := TurnPlanSnapshot{
		Domain:     strings.TrimSpace(domain),
		Mode:       strings.TrimSpace(mode),
		SOP:        sop,
		ToolsAllow: append([]string(nil), toolsAllow...),
	}
	raw, _ := json.Marshal(snap)
	c.Metadata[metaLastTurnPlan] = json.RawMessage(raw)
}

// LastTurnPlanFromSession reads the persisted routing snapshot.
func LastTurnPlanFromSession(c *ChatSession) (TurnPlanSnapshot, bool) {
	if c == nil || c.Metadata == nil {
		return TurnPlanSnapshot{}, false
	}
	raw, ok := c.Metadata[metaLastTurnPlan]
	if !ok || raw == nil {
		return TurnPlanSnapshot{}, false
	}
	switch v := raw.(type) {
	case json.RawMessage:
		var snap TurnPlanSnapshot
		if err := json.Unmarshal(v, &snap); err != nil {
			return TurnPlanSnapshot{}, false
		}
		return snap, snap.Domain != ""
	case map[string]any:
		b, err := json.Marshal(v)
		if err != nil {
			return TurnPlanSnapshot{}, false
		}
		var snap TurnPlanSnapshot
		if err := json.Unmarshal(b, &snap); err != nil {
			return TurnPlanSnapshot{}, false
		}
		return snap, snap.Domain != ""
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return TurnPlanSnapshot{}, false
		}
		var snap TurnPlanSnapshot
		if err := json.Unmarshal(b, &snap); err != nil {
			return TurnPlanSnapshot{}, false
		}
		return snap, snap.Domain != ""
	}
}

// SyncLastTurnToolsCalled stores tool names invoked on the latest completed turn.
func (c *ChatSession) SyncLastTurnToolsCalled(names []string) {
	if c == nil {
		return
	}
	if c.Metadata == nil {
		c.Metadata = map[string]any{}
	}
	out := append([]string(nil), names...)
	c.Metadata[metaLastTurnToolsCalls] = out
}

// LastTurnToolsCalledFromSession returns tool names recorded for the latest turn.
func LastTurnToolsCalledFromSession(c *ChatSession) []string {
	if c == nil || c.Metadata == nil {
		return nil
	}
	raw, ok := c.Metadata[metaLastTurnToolsCalls]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return append([]string(nil), v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		var out []string
		if err := json.Unmarshal(b, &out); err != nil {
			return nil
		}
		return out
	}
}

// ToolsCalledFromStepRecords extracts unique tool names from one turn's records.
func ToolsCalledFromStepRecords(records []ChatStepRecord) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, rec := range records {
		if rec.Kind != "tool" || strings.TrimSpace(rec.ToolName) == "" {
			continue
		}
		name := strings.TrimSpace(rec.ToolName)
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}
