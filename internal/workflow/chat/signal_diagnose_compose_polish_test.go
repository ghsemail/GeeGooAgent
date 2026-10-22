package chat

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func TestFinalizeComposeTrimsOnDeadline(t *testing.T) {
	flow := &Flow{}
	composeCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	time.Sleep(5 * time.Millisecond)
	text, ok := finalizeLLMComposeAnswer(flow, "完整一句。半句未完以及严格", nil, context.DeadlineExceeded, composeCtx)
	if !ok {
		t.Fatal("expected partial")
	}
	if text != "完整一句。" {
		t.Fatalf("got %q", text)
	}
}

func TestFinalizeComposeRendersSectionMarkers(t *testing.T) {
	flow := &Flow{}
	in := llm.RenderComposeSectionDocument(llm.ParseComposeSectionBodies("[[S1]]在腾讯上 probe 多次触发。"))
	text, ok := finalizeLLMComposeAnswer(flow, in, nil, nil, context.Background())
	if !ok {
		t.Fatal("expected ok")
	}
	if !strings.Contains(text, "### 一、信号触发情况") {
		t.Fatalf("got %q", text)
	}
}
