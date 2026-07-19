package runtime

import (
	"encoding/json"
	"time"
)

// AgentEventSchemaVersion is the NDJSON agent progress schema version.
const AgentEventSchemaVersion = 1

// AgentEvent is one line of machine-readable agent loop progress (NDJSON).
type AgentEvent struct {
	SchemaVersion int            `json:"schema_version"`
	Event         string         `json:"event"`
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

// ProgressToAgentEvent maps legacy EmitProgress names to AgentEvent (passthrough data).
func ProgressToAgentEvent(event string, data map[string]any) AgentEvent {
	return NewAgentEvent(event, data)
}
