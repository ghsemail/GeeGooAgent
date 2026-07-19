package chatui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestWideLogoDisplayWidth(t *testing.T) {
	w := wideLogoDisplayWidth()
	if w < 45 {
		t.Fatalf("logo too narrow: %d", w)
	}
}

func TestShouldShowWideLogo(t *testing.T) {
	logoW := wideLogoDisplayWidth()
	if !shouldShowWideLogo(logoW + 2) {
		t.Fatal("expected wide logo when terminal fits")
	}
	if shouldShowWideLogo(logoW - 1) {
		t.Fatal("expected hero fallback when terminal too narrow")
	}
	if !shouldShowWideLogo(80) {
		t.Fatalf("expected wide logo on 80 cols (logoW=%d)", logoW)
	}
}

func TestRenderWideLogoForWidthCentered(t *testing.T) {
	termW := wideLogoDisplayWidth() + 10
	out := stripANSI(renderWideLogoForWidth(termW))
	lines := strings.Split(out, "\n")
	if len(lines) != len(wideLogoLines) {
		t.Fatalf("lines=%d", len(lines))
	}
	first := lines[0]
	if !strings.HasPrefix(first, " ") {
		t.Fatalf("expected centered logo, got %q", first)
	}
	if lipgloss.Width(first) > termW {
		t.Fatalf("logo wider than terminal: %d", lipgloss.Width(first))
	}
}

func TestRenderWideLogoFallsBackOnNarrowTerminal(t *testing.T) {
	out := renderWideLogoForWidth(wideLogoDisplayWidth() - 1)
	if !strings.Contains(out, "╔═╗") {
		t.Fatalf("expected compact hero on narrow terminal: %q", stripANSI(out))
	}
}

func TestWideLogoBottomRowSpacing(t *testing.T) {
	bottom := wideLogoLines[len(wideLogoLines)-2]
	if !strings.HasPrefix(bottom, " ╚") {
		t.Fatalf("bottom row should align with top row: %q", bottom)
	}
	if strings.Contains(bottom, "╔╝████") {
		t.Fatalf("bottom row blocks should be spaced: %q", bottom)
	}
}
