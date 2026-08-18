package feishu

import (
	"testing"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

func TestReceiveIDType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		id   string
		want string
	}{
		{"oc_abc123", larkim.CreateMessageV1ReceiveIDTypeChatId},
		{"ou_aa1b0b12", larkim.CreateMessageV1ReceiveIDTypeOpenId},
		{"on_union", larkim.CreateMessageV1ReceiveIDTypeUnionId},
		{"user@example.com", larkim.CreateMessageV1ReceiveIDTypeEmail},
		{"  ou_x  ", larkim.CreateMessageV1ReceiveIDTypeOpenId},
	}
	for _, tc := range cases {
		if got := receiveIDType(tc.id); got != tc.want {
			t.Fatalf("receiveIDType(%q)=%q want %q", tc.id, got, tc.want)
		}
	}
}
