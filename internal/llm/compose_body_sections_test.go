package llm

import (
	"strings"
	"testing"
)

func TestRenderComposeSectionGluedProbeTitle(t *testing.T) {
	raw := "[[S1]]腾讯信号多次触发。\n[[S2]]probe结果为买5卖4。\n[[S3]]风险可控。\n[[S4]]综合评价良好。"
	doc := RenderComposeSectionDocument(ParseComposeSectionBodies(raw))
	if strings.Contains(doc, "[[S") {
		t.Fatalf("delimiters leaked: %q", doc)
	}
	if !strings.Contains(doc, "### 二、probe 与命中率") || !strings.Contains(doc, "probe结果为") {
		t.Fatalf("got %q", doc)
	}
	if strings.Contains(doc, "命中率probe") {
		t.Fatalf("title glued to body: %q", doc)
	}
}

func TestRenderComposeNoMarkersFallsBackSection1(t *testing.T) {
	raw := "整段无分隔符的正文。"
	doc := RenderComposeSectionDocument(ParseComposeSectionBodies(raw))
	if !strings.Contains(doc, "### 一、信号触发情况") || !strings.Contains(doc, "整段无分隔符") {
		t.Fatalf("got %q", doc)
	}
}

func TestComposeSectionRendererStreaming(t *testing.T) {
	var out string
	r := ComposeSectionRenderer{OnRendered: func(c string) { out += c }}
	r.PushModelText("[[S1]]首段")
	r.PushModelText("继续\n[[S2]]第二")
	if !strings.Contains(out, "### 一、") || strings.Contains(out, "[[S") {
		t.Fatalf("stream out=%q full=%q", out, r.Document())
	}
}
