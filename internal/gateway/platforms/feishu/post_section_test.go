package feishu_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/gateway/platforms/feishu"
)

func TestBuildMarkdownPostRowsSplitsSectionHeaders(t *testing.T) {
	t.Parallel()
	src := "## 盘前个股 · HK · 2026-08-18\n\n### 腾讯控股（00700.HK）\n\n- **结论**：看空\n\n#### 摘要\n\n腾讯盘前研判看空。\n\n#### 判定依据\n\n- 周线跌破均线"
	rows := feishu.BuildMarkdownPostRows(src)
	if len(rows) < 3 {
		t.Fatalf("want >=3 rows, got %d", len(rows))
	}
	if !strings.Contains(rows[0][0]["text"], "## 盘前个股") {
		t.Fatalf("lead row=%v", rows[0])
	}
	if !strings.HasPrefix(rows[1][0]["text"], "#### 摘要") {
		t.Fatalf("summary row=%v", rows[1])
	}
}
