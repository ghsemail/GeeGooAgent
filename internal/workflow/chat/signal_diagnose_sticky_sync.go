package chat

import (
	"encoding/json"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func syncStickyPanelOpsToChat(chat *chatsession.ChatSession, rt *runtime.Session) {
	if chat == nil || rt == nil || rt.StickyPanelOps == nil {
		return
	}
	if chat.Metadata == nil {
		chat.Metadata = map[string]any{}
	}
	raw, err := json.Marshal(rt.StickyPanelOps)
	if err != nil {
		return
	}
	chat.Metadata[MetaKeyStickyPanelOps] = json.RawMessage(raw)
}

func loadStickyPanelOpsFromChat(chat *chatsession.ChatSession, rt *runtime.Session) {
	if chat == nil || rt == nil || chat.Metadata == nil {
		return
	}
	raw, ok := chat.Metadata[MetaKeyStickyPanelOps]
	if !ok || raw == nil {
		raw, ok = chat.Metadata[metaKeyLegacyStickyPanelOps]
		if !ok || raw == nil {
			return
		}
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
	rt.StickyPanelOps = &o
}
