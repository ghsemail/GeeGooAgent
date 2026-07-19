package chattui

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatui"
	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestFixedWelcomeBannerLayout(t *testing.T) {
	m := NewModel(config.DisplayConfig{}, &LiveSlot{Status: "ready"}, nil)
	m.width = 100
	m.height = 40
	m.rebuildBanner()
	if !m.showWelcomeBanner() {
		t.Fatal("expected welcome banner")
	}
	if !m.canFixWelcomeBanner() {
		t.Fatalf("banner lines=%d should fit height=%d", lineCount(m.banner), m.height)
	}
	m.layoutViewport()
	bannerLines := m.fixedWelcomeBannerLines()
	if bannerLines <= 0 {
		t.Fatal("expected fixed banner lines")
	}
	if m.vp.Height != 40-m.footerLineCount()-bannerLines {
		t.Fatalf("vp height=%d want %d (bannerLines=%d incl top pad)", m.vp.Height, 40-m.footerLineCount()-bannerLines, bannerLines)
	}
	m.refreshViewport()
	if strings.Contains(m.renderTranscript(), "Welcome to GeeGoo Agent") {
		t.Fatal("welcome text should be outside viewport content")
	}
	if !strings.Contains(m.banner, "Welcome to GeeGoo Agent") {
		t.Fatal("welcome text should remain in banner")
	}
	if !strings.Contains(m.banner, "██████") {
		t.Fatal("logo should remain in banner")
	}
}

func TestWelcomeBannerHiddenAfterUserMessage(t *testing.T) {
	m := NewModel(config.DisplayConfig{}, &LiveSlot{
		Status: "ready",
		Blocks: []Block{{Kind: KindUser, Body: "hi"}},
	}, nil)
	m.rebuildBanner()
	if m.showWelcomeBanner() {
		t.Fatal("banner should hide after user message")
	}
}

func TestRenderTranscriptOmitsFixedBanner(t *testing.T) {
	m := NewModel(config.DisplayConfig{}, &LiveSlot{Status: "ready"}, nil)
	m.width = 100
	m.height = 40
	m.rebuildBanner()
	out := m.renderTranscript()
	if strings.Contains(out, chatui.RenderWelcomeTips()) {
		t.Fatal("fixed banner should not be in transcript")
	}
}
