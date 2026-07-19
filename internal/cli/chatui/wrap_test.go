package chatui

import (
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
)

func TestWrapPlain_Chinese(t *testing.T) {
	in := "这是一段很长的中文说明文字用于测试在终端里是否会按显示宽度强制折行而不是挤成一行"
	out := WrapPlain(in, 20)
	if !strings.Contains(out, "\n") {
		t.Fatalf("expected wrapped lines: %q", out)
	}
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		if w := runewidth.StringWidth(line); w > 20 {
			t.Fatalf("line too wide (%d): %q", w, line)
		}
	}
}

func TestWrapPlain_PreservesParagraphs(t *testing.T) {
	in := "第一段。\n\n第二段。"
	out := WrapPlain(in, 40)
	if !strings.Contains(out, "\n\n") {
		t.Fatalf("paragraph break lost: %q", out)
	}
}

func TestWrapWithPrefix_Option(t *testing.T) {
	in := "这是一个很长的选项说明需要折行显示让用户能完整阅读"
	prefix := "  [A] "
	out := WrapWithPrefix(in, prefix, "", 30)
	if !strings.Contains(out, "\n") {
		t.Fatalf("expected wrap: %q", out)
	}
	if !strings.HasPrefix(out, prefix) {
		t.Fatalf("missing prefix: %q", out)
	}
}

func TestContentWrapWidth(t *testing.T) {
	if got := ContentWrapWidth(80); got != 76 {
		t.Fatalf("got %d", got)
	}
	if got := ContentWrapWidth(200); got != 196 {
		t.Fatalf("wide terminal: got %d", got)
	}
}