package runtimeapi

import "testing"

func TestCompareStorePathPerUser(t *testing.T) {
	h := &Handler{App: nil}
	global := h.compareStorePath("")
	user := h.compareStorePath("user/1")
	if global == user {
		t.Fatalf("global and user paths should differ: %s", global)
	}
	if global == "" || user == "" {
		t.Fatal("paths must not be empty")
	}
}

func TestGatewayForCompareSpecOverridesModel(t *testing.T) {
	provider, model := splitModelSpec("deepseek:deepseek-chat")
	if provider != "deepseek" || model != "deepseek-chat" {
		t.Fatalf("split=%s %s", provider, model)
	}
}
