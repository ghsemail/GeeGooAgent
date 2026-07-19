package chatcmd

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestApplyStreamDefaultOff(t *testing.T) {
	d := config.DisplayConfig{}
	res := ApplyStream(&d, nil)
	if !res.OK {
		t.Fatalf("status: %+v", res)
	}
	if d.StreamReplyEnabled() {
		t.Fatal("stream should default off")
	}
	if res.Message == "" {
		t.Fatal("expected status message")
	}
}

func TestApplyStreamOnOff(t *testing.T) {
	d := config.DisplayConfig{}
	res := ApplyStream(&d, []string{"on"})
	if !res.OK || !res.Persist || res.Display == nil || !res.Display.StreamReplyEnabled() {
		t.Fatalf("on: %+v", res)
	}
	d = *res.Display
	res = ApplyStream(&d, []string{"off"})
	if !res.OK || res.Display == nil || res.Display.StreamReplyEnabled() {
		t.Fatalf("off: %+v", res)
	}
}

func TestApplyReplyFormat(t *testing.T) {
	d := config.DisplayConfig{}
	res := ApplyReplyFormat(&d, []string{"plain"})
	if !res.OK || res.Display == nil || res.Display.ReplyMarkdownEnabled() {
		t.Fatalf("plain: %+v", res)
	}
	d = *res.Display
	res = ApplyReplyFormat(&d, []string{"markdown"})
	if !res.OK || res.Display == nil || !res.Display.ReplyMarkdownEnabled() {
		t.Fatalf("markdown: %+v", res)
	}
}
