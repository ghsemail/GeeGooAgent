package chattui

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatui"
)

func TestWriteSegmentDivider(t *testing.T) {
	var m Model
	width := 80
	var b strings.Builder

	b.Reset()
	m.writeSegmentDivider(&b, width, segmentUser, segmentProcess)
	if !strings.Contains(stripANSI(b.String()), "─") {
		t.Fatalf("expected soft divider user→process: %q", b.String())
	}

	b.Reset()
	m.writeSegmentDivider(&b, width, segmentProcess, segmentProcess)
	if !strings.Contains(stripANSI(b.String()), "─") {
		t.Fatalf("expected soft divider thinking→tools: %q", b.String())
	}

	b.Reset()
	m.writeSegmentDivider(&b, width, segmentProcess, segmentReply)
	out := stripANSI(b.String())
	if strings.Contains(out, strings.Repeat("─", 80)) {
		t.Fatalf("should not render gold rule before reply: %q", out)
	}
	if !strings.Contains(out, "⚕ GeeGoo") {
		t.Fatalf("expected agent header before reply: %q", out)
	}

	b.Reset()
	m.writeSegmentDivider(&b, width, segmentReply, segmentUser)
	if !strings.Contains(stripANSI(b.String()), strings.Repeat("─", 10)) {
		t.Fatalf("expected gold rule before next user turn: %q", b.String())
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
