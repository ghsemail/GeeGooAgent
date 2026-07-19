package chattui

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatui"
)

func TestWriteSegmentDivider(t *testing.T) {
	var m Model
	var b strings.Builder
	m.writeSegmentDivider(&b, segmentUser, segmentProcess)
	if b.String() != "" {
		t.Fatalf("no divider before process panel: %q", b.String())
	}
	b.Reset()
	m.writeSegmentDivider(&b, segmentUser, segmentReply)
	if b.String() != "" {
		t.Fatalf("no divider between user and reply: %q", b.String())
	}
	b.Reset()
	m.writeSegmentDivider(&b, segmentProcess, segmentReply)
	if b.String() != "\n" {
		t.Fatalf("expected blank line after process: %q", b.String())
	}
	b.Reset()
	m.writeSegmentDivider(&b, segmentReply, segmentUser)
	if b.String() != "\n" {
		t.Fatalf("expected blank line before new user: %q", b.String())
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
}
