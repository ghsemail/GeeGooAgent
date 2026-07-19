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
	if !strings.Contains(plain, "[A]") || !strings.Contains(plain, "B方案") {
		t.Fatalf("panel=%q", plain)
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
