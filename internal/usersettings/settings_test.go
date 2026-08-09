package usersettings

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestApply_feishuGateway(t *testing.T) {
	base := config.LLMConfig{Model: "base", CatalogModelID: "ops-default"}
	doc := &Doc{
		CatalogModelID: "global-user",
		Gateways: map[string]GatewayEntry{
			"feishu": {CatalogModelID: "feishu-model"},
		},
	}
	got := Apply(base, doc, "lark")
	if got.CatalogModelID != "feishu-model" {
		t.Fatalf("got %#v", got)
	}
}
