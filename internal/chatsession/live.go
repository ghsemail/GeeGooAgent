package chatsession

import (
	"sync"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/infra"
)

const liveStateKeyPrefix = "chat_live/"
const maxLiveEvents = 40

// LiveEvent is one in-flight progress event for remote debugging.
type LiveEvent struct {
	Event string         `json:"event"`
	Data  map[string]any `json:"data,omitempty"`
	At    time.Time      `json:"at"`
}

// LiveSessionState tracks an active chat turn across processes (CLI ↔ agent-runtime).
type LiveSessionState struct {
	SessionID string      `json:"session_id"`
	Busy      bool        `json:"busy"`
	Status    string      `json:"status"`
	LastEvent string      `json:"last_event"`
	UpdatedAt time.Time   `json:"updated_at"`
	Error     string      `json:"error,omitempty"`
	Events    []LiveEvent `json:"events,omitempty"`
}

// LivePublisher writes live session progress to the shared state store.
type LivePublisher struct {
	store     *infra.StateStore
	sessionID string
	mu        sync.Mutex
}

// NewLivePublisher creates a cross-process live progress writer.
func NewLivePublisher(store *infra.StateStore, sessionID string) *LivePublisher {
	if store == nil || sessionID == "" {
		return nil
	}
	return &LivePublisher{store: store, sessionID: sessionID}
}

// LoadLiveState reads the current live state for a session.
func LoadLiveState(store *infra.StateStore, sessionID string) (*LiveSessionState, error) {
	if store == nil || sessionID == "" {
		return nil, nil
	}
	data, err := store.Load(liveStateKeyPrefix + sessionID)
	if err != nil || data == nil {
		return nil, err
	}
	return liveStateFromMap(data), nil
}

// Emit records a progress event and persists it for remote SSE consumers.
func (p *LivePublisher) Emit(event string, data map[string]any) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	state, _ := LoadLiveState(p.store, p.sessionID)
	if state == nil {
		state = &LiveSessionState{SessionID: p.sessionID}
	}
	state.LastEvent = event
	state.UpdatedAt = time.Now().UTC()
	applyLiveEvent(state, event, data)
	_ = p.store.Save(liveStateKeyPrefix+p.sessionID, state.toMap())
}

// EndTurn marks the session idle after a turn completes.
func (p *LivePublisher) EndTurn() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	state, _ := LoadLiveState(p.store, p.sessionID)
	if state == nil {
		state = &LiveSessionState{SessionID: p.sessionID}
	}
	state.Busy = false
	state.Status = "ready"
	state.LastEvent = "turn_end"
	state.UpdatedAt = time.Now().UTC()
	_ = p.store.Save(liveStateKeyPrefix+p.sessionID, state.toMap())
}

func applyLiveEvent(state *LiveSessionState, event string, data map[string]any) {
	if state == nil {
		return
	}
	if data == nil {
		data = map[string]any{}
	}
	switch event {
	case "turn_start":
		state.Busy = true
		state.Status = "thinking"
		state.Error = ""
	case "stream_delta":
		if state.Busy {
			state.Status = "streaming"
		}
	case "llm_plan":
		state.Status = "planning"
	case "llm_tools", "tool_start":
		state.Status = "tool"
	case "reply_start":
		state.Status = "replying"
	case "error":
		state.Busy = false
		state.Status = "error"
		if msg, _ := data["message"].(string); msg != "" {
			state.Error = msg
		}
	case "turn_end":
		state.Busy = false
		state.Status = "ready"
	}
	state.Events = append(state.Events, LiveEvent{Event: event, Data: cloneLiveData(data), At: time.Now().UTC()})
	if len(state.Events) > maxLiveEvents {
		state.Events = state.Events[len(state.Events)-maxLiveEvents:]
	}
}

func cloneLiveData(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (s *LiveSessionState) toMap() map[string]any {
	events := make([]map[string]any, 0, len(s.Events))
	for _, ev := range s.Events {
		item := map[string]any{"event": ev.Event, "at": ev.At.Format(time.RFC3339Nano)}
		if len(ev.Data) > 0 {
			item["data"] = ev.Data
		}
		events = append(events, item)
	}
	return map[string]any{
		"session_id": s.SessionID,
		"busy":       s.Busy,
		"status":     s.Status,
		"last_event": s.LastEvent,
		"updated_at": s.UpdatedAt.Format(time.RFC3339Nano),
		"error":      s.Error,
		"events":     events,
	}
}

func liveStateFromMap(data map[string]any) *LiveSessionState {
	state := &LiveSessionState{
		SessionID: stringField(data, "session_id"),
		Status:    stringField(data, "status"),
		LastEvent: stringField(data, "last_event"),
		Error:     stringField(data, "error"),
	}
	if v, ok := data["busy"].(bool); ok {
		state.Busy = v
	}
	if t, err := time.Parse(time.RFC3339Nano, stringField(data, "updated_at")); err == nil {
		state.UpdatedAt = t
	}
	if raw, ok := data["events"].([]any); ok {
		for _, item := range raw {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			ev := LiveEvent{Event: stringField(m, "event")}
			if d, ok := m["data"].(map[string]any); ok {
				ev.Data = d
			}
			if t, err := time.Parse(time.RFC3339Nano, stringField(m, "at")); err == nil {
				ev.At = t
			}
			state.Events = append(state.Events, ev)
		}
	}
	return state
}
