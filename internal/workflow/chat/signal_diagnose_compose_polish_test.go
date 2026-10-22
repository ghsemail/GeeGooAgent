package chat

import (
	"context"
	"testing"
	"time"

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

func TestFinalizeComposeNormalizesSectionHeading(t *testing.T) {
	flow := &Flow{}
	in := "### 一、信号触发情况在腾讯上 probe 多次触发。"
	text, ok := finalizeLLMComposeAnswer(flow, in, nil, nil, context.Background())
	if !ok {
		t.Fatal("expected ok")
	}
	if text == in {
		t.Fatalf("expected section split, got %q", text)
	}
}
