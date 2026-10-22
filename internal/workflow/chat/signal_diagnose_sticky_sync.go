package chat

import (
	"encoding/json"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func syncSignalDiagnoseStickyOptsToChat(chat *chatsession.ChatSession, rt *runtime.Session) {
	if chat == nil || rt == nil || rt.SignalDiagnoseStickyOpts == nil {
		return
	}
	if chat.Metadata == nil {
		chat.Metadata = map[string]any{}
	}
	raw, err := json.Marshal(rt.SignalDiagnoseStickyOpts)
	if err != nil {
		return
	}
	chat.Metadata[MetaKeySignalDiagnoseStickyOpts] = json.RawMessage(raw)
}

func loadSignalDiagnoseStickyOptsFromChat(chat *chatsession.ChatSession, rt *runtime.Session) {
	if chat == nil || rt == nil || chat.Metadata == nil {
		return
	}
	raw, ok := chat.Metadata[MetaKeySignalDiagnoseStickyOpts]
	if !ok || raw == nil {
		return
	}
	var o runtime.SignalDiagnoseOpts
	switch v := raw.(type) {
	case json.RawMessage:
		if err := json.Unmarshal(v, &o); err != nil {
			return
		}
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return
		}
		if err := json.Unmarshal(b, &o); err != nil {
			return
		}
	}
	rt.SignalDiagnoseStickyOpts = &o
}
