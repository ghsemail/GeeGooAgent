package chat

import (
	"context"
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

func TestFinalizeComposeNormalizesHeading(t *testing.T) {
	flow := &Flow{}
	in := "###信号诊断评价本次策略表现稳定。"
	text, ok := finalizeLLMComposeAnswer(flow, in, nil, nil, context.Background())
	if !ok {
		t.Fatal("expected ok")
	}
	if text == in {
		t.Fatalf("expected heading normalized, got %q", text)
	}
	if !llm.IsMostlyChinese(text) {
		t.Fatalf("not chinese: %q", text)
	}
}
