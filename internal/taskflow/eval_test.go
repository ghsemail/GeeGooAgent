package taskflow

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func TestEvalScenarioMultiStrategyCompare(t *testing.T) {
	session := runtime.NewSession()
	msg := "帮我在腾讯上对比 Macd4H 和 共振的信号"
	if !ShouldStartMultiStrategyFlow(msg, session, nil) {
		t.Fatal("expected taskflow start for multi-strategy compare message")
	}
	flow := newMultiStrategyFlow(msg, session)
	if flow.Template != TemplateMultiStrategyCompare {
		t.Fatalf("template=%s", flow.Template)
	}
	if flow.StockQuery == "" {
		t.Fatal("expected stock query 腾讯")
	}
	if len(flow.Strategies) < 2 {
		t.Fatalf("strategies=%v", flow.Strategies)
	}
}

func TestEvalScenarioContinueAll(t *testing.T) {
	session := runtime.NewSession()
	session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: "测一下腾讯 Macd4H 买卖点"})
	if !ShouldStartMultiStrategyFlow("你挨个跑一下", session, nil) {
		t.Fatal("expected continue-all to start taskflow")
	}
}
