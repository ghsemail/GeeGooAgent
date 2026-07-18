package chattui

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatui"
)

func TestWriteSegmentDivider(t *testing.T) {
	var m Model
	var b strings.Builder
	m.writeSegmentDivider(&b, 80, segmentUser, segmentProcess)
	if b.String() != "" {
		t.Fatalf("no divider before process panel: %q", b.String())
	}
	b.Reset()
	m.writeSegmentDivider(&b, 80, segmentUser, segmentReply)
	if !strings.Contains(b.String(), "─") {
		t.Fatalf("expected soft divider between user and reply: %q", b.String())
	}
	b.Reset()
	m.writeSegmentDivider(&b, 80, segmentReply, segmentUser)
	out := b.String()
	if out == "" || !strings.Contains(stripANSI(out), strings.Repeat("─", 40)) {
		t.Fatalf("expected turn rule before new user: %q", out)
	}
}

func stripANSI(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		out.WriteByte(s[i])
	}
	return out.String()
}

func TestRenderSoftDividerShorterThanRule(t *testing.T) {
	soft := stripANSI(chatui.RenderSoftDivider(80))
	rule := stripANSI(chatui.RenderRule(80))
	if len(soft) >= len(rule) {
		t.Fatalf("soft=%d rule=%d", len(soft), len(rule))
	}
	if !strings.HasPrefix(strings.TrimSpace(soft), "──") {
		t.Fatalf("soft=%q", soft)
	}
}
