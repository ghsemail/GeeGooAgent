package diagram

import (
	"encoding/json"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/skills"
)

func TestParsePhaseTitles(t *testing.T) {
	md := `
| Phase | 说明 |
| ` + "`cognition_pick`" + ` | 解析策略名称 |
| ` + "`cognition_save_kb`" + ` | 写入知识库 |

1. ` + "`resolve_symbol`" + ` — 解析标的
`
	got := parsePhaseTitles(md)
	if got["cognition_pick"] != "解析策略名称" {
		t.Fatalf("cognition_pick=%q", got["cognition_pick"])
	}
	if got["resolve_symbol"] != "解析标的" {
		t.Fatalf("resolve_symbol=%q", got["resolve_symbol"])
	}
}

func TestCompileShowsChineseAndParams(t *testing.T) {
	spec, ok := skills.Default().Get("generate_strategy_cognition")
	if !ok {
		t.Fatal("missing skill")
	}
	doc, err := compileSkill(spec)
	if err != nil {
		t.Fatal(err)
	}
	var ir workflowIR
	if err := json.Unmarshal(doc.IR, &ir); err != nil {
		t.Fatal(err)
	}
	var pick *workflowNode
	for i := range ir.Nodes {
		if ir.Nodes[i].ID == "cognition_pick" || ir.Nodes[i].Label == "解析策略名称" {
			pick = &ir.Nodes[i]
			break
		}
	}
	if pick == nil {
		t.Fatalf("nodes=%v", nodeLabels(doc))
	}
	if pick.Label != "解析策略名称" {
		t.Fatalf("label=%q", pick.Label)
	}
	if pick.Sublabel == "" || pick.Sublabel == pick.Label {
		t.Fatalf("sublabel=%q", pick.Sublabel)
	}
}

func TestFormatArgs(t *testing.T) {
	got := formatArgs(map[string]any{"code": "000001.SH", "limit": 8, "nested": map[string]any{"x": 1}})
	if got != "code=000001.SH · limit=8" {
		t.Fatalf("got %q", got)
	}
}
