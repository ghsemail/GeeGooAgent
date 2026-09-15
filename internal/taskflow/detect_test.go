package taskflow

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func TestShouldStartAfterProbeSession(t *testing.T) {
	session := runtime.NewSession()
	session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: "测腾讯 Macd4H 买卖点"})
	if !ShouldStartMultiStrategyFlow("你挨个跑一下", session, nil) {
		t.Fatal("expected start on 挨个跑 with prior stock context")
	}
}

func TestShouldNotStartGenericChat(t *testing.T) {
	session := runtime.NewSession()
	if ShouldStartMultiStrategyFlow("你好", session, nil) {
		t.Fatal("expected no start for generic chat")
	}
}
