package chattui

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestLiveBlockCompactPreviewByDefault(t *testing.T) {
	cfg := config.DisplayConfig{DetailsMode: config.ModeCollapsed}
	b := Block{Kind: KindThinking, Live: true, Body: "line1\nline2"}
	if b.IsExpanded(cfg) {
		t.Fatal("live should not fully expand in collapsed mode")
	}
	if !b.ShowLivePreview(cfg) {
		t.Fatal("live should show preview line")
	}
	if b.LastBodyLine() != "line2" {
		t.Fatalf("last line=%q", b.LastBodyLine())
	}
}

func TestLiveBlockExpandedModeShowsFullBody(t *testing.T) {
	cfg := config.DisplayConfig{DetailsMode: config.ModeExpanded}
	b := Block{Kind: KindTools, Live: true, Body: "→ foo"}
	if !b.IsExpanded(cfg) {
		t.Fatal("expanded mode should show full live body")
	}
	if b.ShowLivePreview(cfg) {
		t.Fatal("no preview when fully expanded")
	}
}

func TestHistoryFollowsCollapsed(t *testing.T) {
	cfg := config.DisplayConfig{DetailsMode: config.ModeCollapsed}
	b := Block{Kind: KindThinking, Live: false, Body: "reason"}
	if b.IsExpanded(cfg) {
		t.Fatal("history should collapse")
	}
	if !b.IsVisible(cfg) {
		t.Fatal("collapsed still visible as header")
	}
}

func TestHiddenNotVisible(t *testing.T) {
	cfg := config.DisplayConfig{DetailsMode: config.ModeHidden}
	b := Block{Kind: KindTools, Live: false}
	if b.IsVisible(cfg) {
		t.Fatal("hidden should not show")
	}
}

func TestUserOverride(t *testing.T) {
	cfg := config.DisplayConfig{DetailsMode: config.ModeCollapsed}
	b := Block{Kind: KindThinking, Live: false, Body: "x"}
	b.ToggleExpand(cfg)
	if !b.IsExpanded(cfg) {
		t.Fatal("toggle should expand")
	}
	b.ToggleExpand(cfg)
	if b.IsExpanded(cfg) {
		t.Fatal("toggle should collapse")
	}
}

func TestLiveToggleExpandWithoutClearingLive(t *testing.T) {
	cfg := config.DisplayConfig{DetailsMode: config.ModeCollapsed}
	b := Block{Kind: KindTools, Live: true, Body: "→ foo"}
	b.ToggleExpand(cfg)
	if !b.Live {
		t.Fatal("toggle during live should keep Live=true")
	}
	if !b.IsExpanded(cfg) {
		t.Fatal("toggle should expand live block")
	}
}

func TestReplyAlwaysExpanded(t *testing.T) {
	cfg := config.DisplayConfig{DetailsMode: config.ModeHidden}
	b := Block{Kind: KindReply, Body: "hi"}
	if !b.IsExpanded(cfg) || !b.IsVisible(cfg) {
		t.Fatal("reply always shown")
	}
}
