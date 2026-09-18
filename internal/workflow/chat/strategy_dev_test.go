package chat

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func TestIsStrategyDevIntent(t *testing.T) {
	if !IsStrategyDevIntent("策略认知 Macd4H") {
		t.Fatal("expected strategy dev intent")
	}
	if IsStrategyDevIntent("帮我在腾讯上对比 Macd4H 和 共振") {
		t.Fatal("multi-strategy compare should not be strategy dev")
	}
}

func TestShouldStartStrategyDevFlow(t *testing.T) {
	if !ShouldStartStrategyDevFlow("策略开发 共振", nil) {
		t.Fatal("expected new strategy_dev flow")
	}
	active := &Flow{Template: SkillMultiStrategyCompare, Status: StatusRunning}
	if ShouldStartStrategyDevFlow("策略认知 Macd4H", active) {
		t.Fatal("should not start when another flow is active")
	}
}

func TestExtractStrategyDevQuery(t *testing.T) {
	got := extractStrategyDevQuery("策略认知 Macd4H 和 共振")
	if got == "" {
		t.Fatal("expected query")
	}
}

func TestShouldStartMultiStrategySkipsStrategyDev(t *testing.T) {
	session := runtime.NewSession()
	if ShouldStartMultiStrategyFlow("策略认知 Macd4H", session, nil) {
		t.Fatal("strategy dev phrase should not start multi_strategy")
	}
}
