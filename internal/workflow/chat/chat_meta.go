package chat

import (
	"encoding/json"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

// SyncToChat persists flow state from runtime session into chat metadata.
func SyncToChat(chat *chatsession.ChatSession, rt *runtime.Session) {
	if chat == nil {
		return
	}
	if chat.Metadata == nil {
		chat.Metadata = map[string]any{}
	}
	flow := LoadFromSession(rt)
	if flow == nil {
		delete(chat.Metadata, MetaKeyActiveFlow)
		return
	}
	raw, err := json.Marshal(flow)
	if err != nil {
		return
	}
	chat.Metadata[MetaKeyActiveFlow] = json.RawMessage(raw)
}

// LoadFromChat reconstructs flow state from chat metadata into runtime session.
func LoadFromChat(chat *chatsession.ChatSession, rt *runtime.Session) {
	if chat == nil || rt == nil || chat.Metadata == nil {
		return
	}
	raw, ok := chat.Metadata[MetaKeyActiveFlow]
	if !ok || raw == nil {
		return
	}
	var flow Flow
	switch v := raw.(type) {
	case json.RawMessage:
		if err := json.Unmarshal(v, &flow); err != nil {
			return
		}
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return
		}
		if err := json.Unmarshal(b, &flow); err != nil {
			return
		}
	}
	if flow.RunID == "" {
		return
	}
	normalizeOnLoad(&flow)
	SaveToSession(rt, &flow)
}

// RawFromChat returns metadata blob for API responses.
func RawFromChat(chat *chatsession.ChatSession) any {
	if chat == nil || chat.Metadata == nil {
		return nil
	}
	return chat.Metadata[MetaKeyActiveFlow]
}

func normalizeOnLoad(flow *Flow) {
	if flow == nil {
		return
	}
	if flow.Status == StatusRunning {
		if time.Since(flow.UpdatedAt) > 2*time.Minute {
			MarkInterrupted(flow)
		}
	}
}
