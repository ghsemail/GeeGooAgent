package agent_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
)

func TestDecideRetrievalGate(t *testing.T) {
	if got := agent.DecideRetrievalGate("你好"); got.Decision != "skip" {
		t.Fatalf("greeting: %+v", got)
	}
	if got := agent.DecideRetrievalGate("/think on"); got.Decision != "skip" {
		t.Fatalf("slash: %+v", got)
	}
	if got := agent.DecideRetrievalGate("上次查的腾讯股价多少"); got.Decision != "retrieve" {
		t.Fatalf("history: %+v", got)
	}
	if got := agent.DecideRetrievalGate("00700.HK 现在多少钱"); got.Decision != "retrieve" {
		t.Fatalf("stock: %+v", got)
	}
}
