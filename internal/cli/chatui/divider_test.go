package chatui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderRuleFullWidthGold(t *testing.T) {
	rule := RenderRule(120)
	if lipgloss.Width(rule) != 120 {
		t.Fatalf("rule width: got %d want 120", lipgloss.Width(rule))
	}
	plain := stripANSI(rule)
	if !strings.HasPrefix(plain, strings.Repeat("─", 10)) {
		t.Fatalf("rule=%q", plain)
	}
}

func TestRenderAgentHeader(t *testing.T) {
	header := RenderAgentHeader(80)
	if lipgloss.Width(header) != 80 {
		t.Fatalf("header width: got %d want 80", lipgloss.Width(header))
	}
	plain := stripANSI(header)
	if !strings.HasPrefix(plain, "⚕ GeeGoo ") {
		t.Fatalf("header=%q", plain)
	}
}

func TestRenderStatusBox(t *testing.T) {
	box := RenderStatusBox(60)
	lines := strings.Split(stripANSI(box), "\n")
	if len(lines) != 3 {
		t.Fatalf("lines=%d", len(lines))
	}
	if lines[1] != "Initializing agent..." {
		t.Fatalf("middle=%q", lines[1])
	}
	if lipgloss.Width(lines[0]) != 60 || lipgloss.Width(lines[2]) != 60 {
		t.Fatalf("rule widths: top=%d bottom=%d", lipgloss.Width(lines[0]), lipgloss.Width(lines[2]))
	}
}

func TestRenderSoftDividerShorterThanRule(t *testing.T) {
	soft := stripANSI(RenderSoftDivider(80))
	rule := stripANSI(RenderRule(80))
	if len(soft) >= len(rule) {
		t.Fatalf("soft=%d rule=%d", len(soft), len(rule))
	}
	if !strings.HasPrefix(strings.TrimSpace(soft), "──") {
		t.Fatalf("soft=%q", soft)
	}
}
