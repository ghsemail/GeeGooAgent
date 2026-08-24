package gateway

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestParseClarifyReplyLetter(t *testing.T) {
	choices := []string{"单指标", "组合信号"}
	answer, ok := ParseClarifyReply("A", choices)
	if !ok || answer != "单指标" {
		t.Fatalf("answer=%q ok=%v", answer, ok)
	}
}

func TestParseClarifyReplyCustomText(t *testing.T) {
	choices := []string{"单指标", "组合信号"}
	answer, ok := ParseClarifyReply("我想用 SAR 策略", choices)
	if !ok || answer != "我想用 SAR 策略" {
		t.Fatalf("answer=%q ok=%v", answer, ok)
	}
}

func TestFormatClarifyMessageIncludesOther(t *testing.T) {
	msg := FormatClarifyMessage("选哪种信号？", []string{"单指标", "组合"})
	if !strings.Contains(msg, "其他（自行输入）") {
		t.Fatalf("msg=%q", msg)
	}
}

func TestFormatClarifyOpenEndedInquiry(t *testing.T) {
	msg := FormatClarifyMessage("请说明你的目标", nil)
	if !strings.Contains(msg, "请直接回复") {
		t.Fatalf("msg=%q", msg)
	}
}

func TestIsOtherClarifySelection(t *testing.T) {
	choices := []string{"单指标", "组合信号"}
	if !IsOtherClarifySelection("C", choices) {
		t.Fatal("expected C -> other")
	}
	if IsOtherClarifySelection("A", choices) {
		t.Fatal("expected A -> not other")
	}
}

func TestIsClarifySkip(t *testing.T) {
	if !IsClarifySkip("跳过") {
		t.Fatal("expected skip")
	}
}

func TestParseClarifyReplyOtherDoesNotReturnLetter(t *testing.T) {
	choices := []string{"单指标", "组合信号"}
	answer, ok := ParseClarifyReply("C", choices)
	if ok || answer != "" {
		t.Fatalf("answer=%q ok=%v", answer, ok)
	}
}

func TestClarifyHubDeliverAnswer(t *testing.T) {
	hub := NewClarifyHub()
	key := "feishu:chat:user"
	done := make(chan string, 1)
	go func() {
		answer, ok := hub.Wait(context.Background(), key)
		if ok {
			done <- answer
		}
		close(done)
	}()
	time.Sleep(20 * time.Millisecond)
	if !hub.DeliverAnswer(key, "B") {
		t.Fatal("deliver failed")
	}
	if got := <-done; got != "B" {
		t.Fatalf("got %q", got)
	}
}
