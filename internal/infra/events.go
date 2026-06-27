package infra

import "log"

// EventHandler receives one event payload.
type EventHandler func(payload map[string]any)

// EventRecord is one emitted event.
type EventRecord struct {
	Event   string
	Payload map[string]any
}

// EventBus emits in-process events (Python InProcessEventBus parity).
type EventBus struct {
	handlers map[string][]EventHandler
	History  []EventRecord
}

// NewEventBus creates an empty bus.
func NewEventBus() *EventBus {
	return &EventBus{handlers: make(map[string][]EventHandler)}
}

// On registers a handler for an event name.
func (b *EventBus) On(event string, handler EventHandler) {
	b.handlers[event] = append(b.handlers[event], handler)
}

// Emit dispatches synchronously; handler panics are logged, not propagated.
func (b *EventBus) Emit(event string, payload map[string]any) {
	record := map[string]any{}
	for k, v := range payload {
		record[k] = v
	}
	b.History = append(b.History, EventRecord{Event: event, Payload: record})
	for _, handler := range b.handlers[event] {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("event_handler_failed event=%s err=%v", event, r)
				}
			}()
			handler(record)
		}()
	}
}
