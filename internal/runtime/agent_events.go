package runtime

import (
	"encoding/json"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/runtime/events"
)

// AgentEventSchemaVersion is the NDJSON agent progress schema version.
const AgentEventSchemaVersion = 1

// AgentEvent is one line of machine-readable agent loop progress (NDJSON).
type AgentEvent struct {
	SchemaVersion int            `json:"schema_version"`
	Event         string         `json:"event"`
	ItemType      string         `json:"item_type,omitempty"`
	Ts            string         `json:"ts"`
	Data          map[string]any `json:"data,omitempty"`
}

// NewAgentEvent builds a timestamped progress event.
func NewAgentEvent(name string, data map[string]any) AgentEvent {
	if data == nil {
		data = map[string]any{}
	}
	return AgentEvent{
		SchemaVersion: AgentEventSchemaVersion,
		Event:         name,
		Ts:            time.Now().UTC().Format(time.RFC3339Nano),
		Data:          data,
	}
}

// EncodeLine returns JSON plus newline for NDJSON sinks.
func (e AgentEvent) EncodeLine() ([]byte, error) {
	raw, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// ProgressToAgentEvent maps legacy EmitProgress names to AgentEvent with item_type.
func ProgressToAgentEvent(event string, data map[string]any) AgentEvent {
	e := NewAgentEvent(event, data)
	e.ItemType = events.ItemTypeForEvent(event)
	return e
}

// ProgressPayload builds an SSE/JSON map with schema_version, event, item_type, ts, and nested data.
// Legacy clients keep reading flat fields (e.g. gate.decision); structured clients use item_type + data.
func ProgressPayload(event string, data map[string]any) map[string]any {
	ev := ProgressToAgentEvent(event, data)
	out := map[string]any{
		"schema_version": ev.SchemaVersion,
		"event":          ev.Event,
		"item_type":      ev.ItemType,
		"ts":             ev.Ts,
	}
	for k, v := range ev.Data {
		out[k] = v
	}
	if len(ev.Data) > 0 {
		out["data"] = ev.Data
	}
	return out
}
