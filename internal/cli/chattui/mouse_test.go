package chattui

import (
	"testing"
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
	b := Block{Kind: KindThinking, Title: "💭", Body: "a\nb\nc", Live: true}
	if EstimateBlockHeight(b, true) < 4 {
		t.Fatalf("got %d", EstimateBlockHeight(b, true))
	}
	if EstimateBlockHeight(b, false) != 1 {
		t.Fatalf("collapsed=%d", EstimateBlockHeight(b, false))
	}
}
