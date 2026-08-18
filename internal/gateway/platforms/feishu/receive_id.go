package feishu

import (
	"strings"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// receiveIDType maps a Feishu receive_id value to the Open API receive_id_type.
// Scheduler targets may be a group chat_id (oc_...) or a user open_id (ou_...).
func receiveIDType(id string) string {
	id = strings.TrimSpace(id)
	switch {
	case strings.HasPrefix(id, "ou_"):
		return larkim.CreateMessageV1ReceiveIDTypeOpenId
	case strings.HasPrefix(id, "on_"):
		return larkim.CreateMessageV1ReceiveIDTypeUnionId
	case strings.Contains(id, "@"):
		return larkim.CreateMessageV1ReceiveIDTypeEmail
	default:
		return larkim.CreateMessageV1ReceiveIDTypeChatId
	}
}
