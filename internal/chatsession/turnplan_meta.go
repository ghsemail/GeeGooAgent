package chatsession

import (
	"encoding/json"
	"strings"
)

const (
	metaLastTurnPlan       = "last_turn_plan"
	metaLastTurnToolsCalls = "last_turn_tools_called"
	metaTurnToolsTrace     = "turn_tools_trace"
	metaTurnPlanTrace      = "turn_plan_trace"
)

// TurnPlanTraceEntry records routing snapshot for one user turn (1-based index).
type TurnPlanTraceEntry struct {
	Turn   int              `json:"turn"`
	Plan   TurnPlanSnapshot `json:"plan"`
}

// TurnPlanSnapshot is stored on ChatSession.Metadata after each agent turn.
type TurnPlanSnapshot struct {
	Domain     string   `json:"domain"`
	Mode       string   `json:"mode"`
	Act        string   `json:"act,omitempty"`
	SOP        bool     `json:"sop"`
	ToolsAllow []string `json:"tools_allow,omitempty"`
}

// TurnToolsEntry records tools invoked during one user turn (1-based index).
type TurnToolsEntry struct {
	Turn  int      `json:"turn"`
	Tools []string `json:"tools"`
}

// SyncLastTurnPlan writes the routing snapshot for eval verification.
func (c *ChatSession) SyncLastTurnPlan(domain, mode, act string, sop bool, toolsAllow []string) {
	if c == nil {
		return
	}
	if c.Metadata == nil {
		c.Metadata = map[string]any{}
	}
	snap := TurnPlanSnapshot{
		Domain:     strings.TrimSpace(domain),
		Mode:       strings.TrimSpace(mode),
		Act:        strings.TrimSpace(act),
		SOP:        sop,
		ToolsAllow: append([]string(nil), toolsAllow...),
	}
	raw, _ := json.Marshal(snap)
	c.Metadata[metaLastTurnPlan] = json.RawMessage(raw)
	c.AppendTurnPlanTrace(snap)
}

// AppendTurnPlanTrace appends one user-turn routing snapshot for eval intent checks.
func (c *ChatSession) AppendTurnPlanTrace(snap TurnPlanSnapshot) {
	if c == nil {
		return
	}
	if c.Metadata == nil {
		c.Metadata = map[string]any{}
	}
	trace := TurnPlanTraceFromSession(c)
	entry := TurnPlanTraceEntry{
		Turn: len(trace) + 1,
		Plan: snap,
	}
	trace = append(trace, entry)
	c.Metadata[metaTurnPlanTrace] = trace
}

// TurnPlanTraceFromSession returns per-turn routing history.
func TurnPlanTraceFromSession(c *ChatSession) []TurnPlanTraceEntry {
	if c == nil || c.Metadata == nil {
		return nil
	}
	raw, ok := c.Metadata[metaTurnPlanTrace]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []TurnPlanTraceEntry:
		return append([]TurnPlanTraceEntry(nil), v...)
	case []any:
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		var out []TurnPlanTraceEntry
		if err := json.Unmarshal(b, &out); err != nil {
			return nil
		}
		return out
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		var out []TurnPlanTraceEntry
		if err := json.Unmarshal(b, &out); err != nil {
			return nil
		}
		return out
	}
}

// FirstTurnPlanFromSession returns the routing snapshot from the opening user turn.
func FirstTurnPlanFromSession(c *ChatSession) (TurnPlanSnapshot, bool) {
	trace := TurnPlanTraceFromSession(c)
	if len(trace) == 0 {
		return TurnPlanSnapshot{}, false
	}
	snap := trace[0].Plan
	return snap, snap.Domain != ""
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

// AppendTurnToolsTrace appends one user-turn tool record for eval execution checks.
func (c *ChatSession) AppendTurnToolsTrace(names []string) {
	if c == nil {
		return
	}
	if c.Metadata == nil {
		c.Metadata = map[string]any{}
	}
	trace := TurnToolsTraceFromSession(c)
	entry := TurnToolsEntry{
		Turn:  len(trace) + 1,
		Tools: append([]string(nil), names...),
	}
	trace = append(trace, entry)
	c.Metadata[metaTurnToolsTrace] = trace
}

// TurnToolsTraceFromSession returns per-turn tool invocation history.
func TurnToolsTraceFromSession(c *ChatSession) []TurnToolsEntry {
	if c == nil || c.Metadata == nil {
		return nil
	}
	raw, ok := c.Metadata[metaTurnToolsTrace]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []TurnToolsEntry:
		return append([]TurnToolsEntry(nil), v...)
	case []any:
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		var out []TurnToolsEntry
		if err := json.Unmarshal(b, &out); err != nil {
			return nil
		}
		return out
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		var out []TurnToolsEntry
		if err := json.Unmarshal(b, &out); err != nil {
			return nil
		}
		return out
	}
}

// JudgedTurnToolsFromTrace returns tools from the last user turn in the trace.
func JudgedTurnToolsFromTrace(trace []TurnToolsEntry) []string {
	if len(trace) == 0 {
		return nil
	}
	return append([]string(nil), trace[len(trace)-1].Tools...)
}

// SessionToolsFromTrace returns the union of tools across all turns.
func SessionToolsFromTrace(trace []TurnToolsEntry) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, entry := range trace {
		for _, name := range entry.Tools {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	return out
}

// LastTurnToolsCalledFromSession returns tool names recorded for the latest turn.
func LastTurnToolsCalledFromSession(c *ChatSession) []string {
	trace := TurnToolsTraceFromSession(c)
	if len(trace) > 0 {
		return JudgedTurnToolsFromTrace(trace)
	}
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
