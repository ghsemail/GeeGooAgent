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

func TestRenderWideLogoLeftAligned(t *testing.T) {
	out := stripANSI(renderWideLogo())
	lines := strings.Split(out, "\n")
	if len(lines) != len(wideLogoLines) {
		t.Fatalf("lines=%d", len(lines))
	}
	if strings.HasPrefix(lines[0], "   ") {
		t.Fatalf("logo should be left aligned, got %q", lines[0])
	}
	if lipgloss.Width(lines[0]) > wideLogoDisplayWidth() {
		t.Fatalf("logo line wider than expected")
	}
}

func TestRenderBannerLogoFallsBackOnNarrowTerminal(t *testing.T) {
	out := stripANSI(renderBannerLogo(wideLogoDisplayWidth() - 1))
	if !strings.Contains(out, "⚕ GeeGoo Agent") || !strings.Contains(out, "╔═╗") {
		t.Fatalf("expected title + compact hero on narrow terminal: %q", out)
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
