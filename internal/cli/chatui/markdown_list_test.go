package chatui

import "testing"

func TestEnsureListSpacing(t *testing.T) {
	in := "文件 & 代码\n- 读写文件\n- grep 搜索"
	out := ensureListSpacing(in)
	if out == in {
		t.Fatalf("expected blank line before list: %q", out)
	}
}
