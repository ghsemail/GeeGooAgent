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
