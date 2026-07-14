package chattui

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestLiveBlockAlwaysExpanded(t *testing.T) {
	cfg := config.DisplayConfig{DetailsMode: config.ModeCollapsed}
	b := Block{Kind: KindThinking, Live: true, Body: "reason"}
	if !b.IsExpanded(cfg) {
		t.Fatal("live should expand")
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

func TestReplyAlwaysExpanded(t *testing.T) {
	cfg := config.DisplayConfig{DetailsMode: config.ModeHidden}
	b := Block{Kind: KindReply, Body: "hi"}
	if !b.IsExpanded(cfg) || !b.IsVisible(cfg) {
		t.Fatal("reply always shown")
	}
}
