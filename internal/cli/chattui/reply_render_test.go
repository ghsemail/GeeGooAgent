package chattui

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestReplyRendersMarkdownWithStreamOff(t *testing.T) {
	m := NewModel(config.DisplayConfig{}, &LiveSlot{Status: "ready"}, nil)
	m.width = 100
	m.display.Normalize()
	m.slots[0].Blocks = []Block{
		{Kind: KindUser, Body: "你能做什么"},
		{Kind: KindReply, Body: "##你好！###1.股票分析 **实时行情-** 查询价格", Live: false},
	}
	out := stripANSI(m.renderTranscript())
	if strings.Contains(out, "##") || strings.Contains(out, "**") {
		t.Fatalf("expected rendered markdown, got %q", out)
	}
	if !strings.Contains(out, "你好") || !strings.Contains(out, "实时行情") {
		t.Fatalf("missing content: %q", out)
	}
}

func TestUpsertTurnReplyAppendsEachTurn(t *testing.T) {
	s := &LiveSlot{Status: "ready"}
	s.upsertTurnReply("first")
	s.upsertTurnReply("second")
	if len(s.Blocks) != 2 {
		t.Fatalf("blocks=%d", len(s.Blocks))
	}
	if s.Blocks[0].Body != "first" || s.Blocks[1].Body != "second" {
		t.Fatalf("bodies=%q %q", s.Blocks[0].Body, s.Blocks[1].Body)
	}
}

func TestUpsertTurnReplyUpdatesLiveStreamBlock(t *testing.T) {
	s := &LiveSlot{Status: "ready"}
	s.ensureLiveReply()
	s.Blocks[0].Body = "partial"
	s.upsertTurnReply("final answer")
	if len(s.Blocks) != 1 {
		t.Fatalf("blocks=%d", len(s.Blocks))
	}
	if s.Blocks[0].Body != "final answer" || s.Blocks[0].Live {
		t.Fatalf("block=%+v", s.Blocks[0])
	}
}
