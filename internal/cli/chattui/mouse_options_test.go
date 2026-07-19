package chattui

import "testing"

func TestProgramOptionsMouseOffNoCapture(t *testing.T) {
	opts := ProgramOptions("off", true)
	if len(opts) != 1 {
		t.Fatalf("want alt-screen only, got %d opts", len(opts))
	}
}

func TestProgramOptionsNoAltScreen(t *testing.T) {
	opts := ProgramOptions("off", false)
	if len(opts) != 0 {
		t.Fatalf("want no opts without alt screen or mouse, got %d", len(opts))
	}
}

func TestProgramOptionsWheelAddsMouse(t *testing.T) {
	opts := ProgramOptions("wheel", true)
	if len(opts) != 2 {
		t.Fatalf("want alt-screen + mouse, got %d opts", len(opts))
	}
}
