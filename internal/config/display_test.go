package config

import "testing"

func TestEffectiveModeFallsBackToGlobal(t *testing.T) {
	d := DisplayConfig{DetailsMode: "collapsed"}
	if got := d.EffectiveMode("thinking"); got != ModeCollapsed {
		t.Fatalf("got %s", got)
	}
	if got := d.EffectiveMode("tools"); got != ModeCollapsed {
		t.Fatalf("tools got %s", got)
	}
}

func TestEffectiveModeSectionOverride(t *testing.T) {
	d := DisplayConfig{
		DetailsMode: ModeCollapsed,
		Sections:    DisplaySections{Thinking: ModeExpanded, Tools: ModeHidden},
	}
	if got := d.EffectiveMode("thinking"); got != ModeExpanded {
		t.Fatalf("thinking got %s", got)
	}
	if got := d.EffectiveMode("tools"); got != ModeHidden {
		t.Fatalf("tools got %s", got)
	}
}

func TestNormalizeDefaultsActivityHidden(t *testing.T) {
	d := DisplayConfig{}
	d.Normalize()
	if d.DetailsMode != ModeCollapsed {
		t.Fatalf("details_mode=%s", d.DetailsMode)
	}
	if d.Sections.Activity != ModeHidden {
		t.Fatalf("activity=%s", d.Sections.Activity)
	}
	if d.Interface != "tui" {
		t.Fatalf("interface=%s", d.Interface)
	}
}

func TestCycleDetailsMode(t *testing.T) {
	if CycleDetailsMode(ModeHidden) != ModeCollapsed {
		t.Fatal("hidden→collapsed")
	}
	if CycleDetailsMode(ModeCollapsed) != ModeExpanded {
		t.Fatal("collapsed→expanded")
	}
	if CycleDetailsMode(ModeExpanded) != ModeHidden {
		t.Fatal("expanded→hidden")
	}
}

func TestReasoningVisibleDefaultTrue(t *testing.T) {
	d := DisplayConfig{}
	if !d.ReasoningVisible() {
		t.Fatal("expected true")
	}
	f := false
	d.ShowReasoning = &f
	if d.ReasoningVisible() {
		t.Fatal("expected false")
	}
}

func TestStreamReplyDefaultOff(t *testing.T) {
	d := DisplayConfig{}
	d.Normalize()
	if d.StreamReplyEnabled() {
		t.Fatal("stream_reply should default off")
	}
	if !d.ReplyMarkdownEnabled() {
		t.Fatal("reply_format should default markdown")
	}
}

func TestStreamReplyOn(t *testing.T) {
	on := true
	d := DisplayConfig{StreamReply: &on}
	if !d.StreamReplyEnabled() {
		t.Fatal("expected on")
	}
}

func TestMouseTrackingDefaultOff(t *testing.T) {
	d := DisplayConfig{}
	d.Normalize()
	if d.MouseTracking != "off" {
		t.Fatalf("mouse_tracking=%q want off", d.MouseTracking)
	}
}

func TestAltScreenDefaultOn(t *testing.T) {
	d := DisplayConfig{}
	if !d.AltScreenEnabled() {
		t.Fatal("alt_screen should default on")
	}
	off := false
	d.AltScreen = &off
	if d.AltScreenEnabled() {
		t.Fatal("alt_screen should be off when set")
	}
}
