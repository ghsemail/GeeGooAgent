package chattui

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestNormalizeMouseMode(t *testing.T) {
	if NormalizeMouseMode("WHEEL") != "wheel" {
		t.Fatal()
	}
	if NormalizeMouseMode("off") != "off" {
		t.Fatal()
	}
}

func TestCycleMouseMode(t *testing.T) {
	if CycleMouseMode("off") != "wheel" {
		t.Fatal()
	}
	if CycleMouseMode("all") != "off" {
		t.Fatal()
	}
}

func TestEstimateBlockHeight(t *testing.T) {
	cfg := config.DisplayConfig{DetailsMode: config.ModeCollapsed}
	b := Block{Kind: KindThinking, Title: "💭", Body: "a\nb\nc", Live: true}
	if EstimateBlockHeight(b, cfg) != 2 {
		t.Fatalf("live preview=%d", EstimateBlockHeight(b, cfg))
	}
	b.Live = false
	if EstimateBlockHeight(b, cfg) != 1 {
		t.Fatalf("collapsed=%d", EstimateBlockHeight(b, cfg))
	}
	expanded := config.DisplayConfig{DetailsMode: config.ModeExpanded}
	b.Live = true
	if EstimateBlockHeight(b, expanded) < 4 {
		t.Fatalf("expanded live=%d", EstimateBlockHeight(b, expanded))
	}
}
