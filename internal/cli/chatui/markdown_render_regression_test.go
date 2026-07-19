package chatui

import (
	"strings"
	"testing"
)

func TestRenderAssistantMarkdown_NoPreprocessRegression(t *testing.T) {
	const sample = `## 组合信号（共 6 个）

**1. SAR信号配套 MACD 直方图趋势**

- signal_id: ` + "`662d0424c4cee7ffb800d0af`" + `
- 频率: 5m / 60m / daily
- **买入**: SAR 向下跳转 + MACD 直方图为正
- **卖出**: SAR 转向上方

| 信号 | 核心逻辑 | 卖出方式 |
|------|----------|----------|
| SAR+MACD | 趋势共振 | 自动（SAR转向） |
| MACD+SAR | 金叉确认 | 自动（死叉） |

> 小结：选定 signal_id 后可继续生成 DCA 方案。

` + "```" + `
00700.HK
` + "```" + `
`
	out := stripANSI(RenderAssistantBoxWith(sample, 100, AssistantRenderOptions{Markdown: true, Live: false}))
	for _, bad := range []string{"##", "**", "|---|", "`662d0424"} {
		if strings.Contains(out, bad) {
			t.Fatalf("raw markdown marker %q should be rendered away: %q", bad, out)
		}
	}
	for _, want := range []string{"组合信号", "SAR信号配套", "662d0424c4cee7ffb800d0af", "趋势共振", "小结"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %q", want, out)
		}
	}
}

func TestRenderAssistantMarkdown_BoldNotFallingBackToPlain(t *testing.T) {
	in := "结论：**买入** 优于观望。"
	out := stripANSI(RenderAssistantMarkdown(in, 80))
	if strings.Contains(out, "**") {
		t.Fatalf("bold should be rendered, not plain fallback: %q", out)
	}
	if !strings.Contains(out, "买入") {
		t.Fatalf("missing content: %q", out)
	}
}
