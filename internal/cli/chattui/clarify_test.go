package chattui

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatui"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestClarifyDisplayOptionsIncludesOther(t *testing.T) {
	m := Model{clarifyChoices: []string{"单指标", "组合信号"}}
	opts := m.clarifyDisplayOptions()
	if len(opts) != 3 || opts[2] != tools.ClarifyOtherLabel {
		t.Fatalf("opts=%v", opts)
	}
}

func TestRenderClarifyPanelShowsLabels(t *testing.T) {
	out := chatui.RenderClarifyPanel("选哪种？", []string{"A方案", "B方案"}, 0, 80)
	plain := stripANSI(out)
	if !strings.Contains(plain, "? 选哪种？") {
		t.Fatalf("missing title: %q", plain)
	}
	if !strings.Contains(plain, "[A]") || !strings.Contains(plain, "B方案") {
		t.Fatalf("panel=%q", plain)
	}
}

func TestRenderClarifyPanelFocusStablePrefix(t *testing.T) {
	opts := []string{"A方案", "B方案", "C方案"}
	a := stripANSI(chatui.RenderClarifyPanel("选哪种？", opts, 0, 80))
	b := stripANSI(chatui.RenderClarifyPanel("选哪种？", opts, 1, 80))
	lineA0 := strings.Split(a, "\n")[1]
	lineB0 := strings.Split(b, "\n")[1]
	if lineA0 == "" || lineB0 == "" {
		t.Fatal("missing option lines")
	}
	if strings.Index(lineA0, "[A]") != strings.Index(lineB0, "[A]") {
		t.Fatalf("option columns shifted:\nfocus0=%q\nfocus1=%q", lineA0, lineB0)
	}
	if strings.Index(lineA0, "[B]") != strings.Index(lineB0, "[B]") {
		t.Fatalf("option B column shifted:\nfocus0=%q\nfocus1=%q", lineA0, lineB0)
	}
}

func TestRenderClarifyPanelWrapsLongOption(t *testing.T) {
	long := "这是一个很长的选项说明需要折行显示让用户能完整阅读全部内容"
	out := chatui.RenderClarifyPanel("请确认你的选择", []string{long}, 0, 50)
	plain := stripANSI(out)
	if !strings.Contains(plain, "\n") {
		t.Fatalf("expected wrapped option: %q", plain)
	}
}
