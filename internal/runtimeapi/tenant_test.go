package runtimeapi_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/runtimeapi"
)

func TestNormalizeSessionSource_feishu(t *testing.T) {
	if got := runtimeapi.NormalizeSessionSource("feishu"); got != "feishu" {
		t.Fatalf("got %q", got)
	}
	if got := runtimeapi.NormalizeSessionSource("lark"); got != "feishu" {
		t.Fatalf("got %q", got)
	}
	if got := runtimeapi.NormalizeSessionSource("app"); got != "trading_app" {
		t.Fatalf("got %q", got)
	}
}
