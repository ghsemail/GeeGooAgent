package chat

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func TestIsGenerateStrategyCognitionIntent(t *testing.T) {
	if !IsGenerateStrategyCognitionIntent("生成策略认知 Macd4H") {
		t.Fatal("expected generate cognition intent")
	}
	if !IsGenerateStrategyCognitionIntent("策略认知 Macd4H") {
		t.Fatal("expected 策略认知 intent")
	}
	if IsGenerateStrategyCognitionIntent("策略开发 Macd4H") {
		t.Fatal("strategy dev should not match generate")
	}
}

func TestIsStrategyDevIntent(t *testing.T) {
	if !IsStrategyDevIntent("策略开发 Macd4H") {
		t.Fatal("expected strategy dev intent")
	}
	if IsStrategyDevIntent("策略认知 Macd4H") {
		t.Fatal("策略认知 should route to generate workflow")
	}
	if IsStrategyDevIntent("帮我在腾讯上对比 Macd4H 和 共振") {
		t.Fatal("multi-strategy compare should not be strategy dev")
	}
}

func TestShouldStartGenerateStrategyCognitionFlow(t *testing.T) {
	if !ShouldStartGenerateStrategyCognitionFlow("策略认知 共振", nil) {
		t.Fatal("expected new generate flow")
	}
	active := &Flow{Template: SkillMultiStrategyCompare, Status: StatusRunning}
	if ShouldStartGenerateStrategyCognitionFlow("策略认知 Macd4H", active) {
		t.Fatal("should not start when another flow is active")
	}
}

func TestShouldStartStrategyDevFlow(t *testing.T) {
	if !ShouldStartStrategyDevFlow("策略开发 共振", nil) {
		t.Fatal("expected new strategy_dev flow")
	}
	if ShouldStartStrategyDevFlow("策略认知 Macd4H", nil) {
		t.Fatal("策略认知 should not start strategy_dev")
	}
	active := &Flow{Template: SkillMultiStrategyCompare, Status: StatusRunning}
	if ShouldStartStrategyDevFlow("策略开发 Macd4H", active) {
		t.Fatal("should not start when another flow is active")
	}
}

func TestExtractStrategyQuery(t *testing.T) {
	got := extractStrategyQuery("生成策略认知 Macd4H")
	if got == "" || got == "生成策略认知 Macd4H" {
		t.Fatalf("unexpected query: %q", got)
	}
	got = extractStrategyQuery("策略开发 共振")
	if got == "" {
		t.Fatal("expected query")
	}
}

func TestShouldStartMultiStrategySkipsStrategyWorkflows(t *testing.T) {
	session := runtime.NewSession()
	if ShouldStartMultiStrategyFlow("策略认知 Macd4H", session, nil) {
		t.Fatal("策略认知 should not start multi_strategy")
	}
	if ShouldStartMultiStrategyFlow("策略开发 Macd4H", session, nil) {
		t.Fatal("策略开发 should not start multi_strategy")
	}
}

func TestNewGenerateFlowTemplate(t *testing.T) {
	f := newGenerateStrategyCognitionFlow("策略认知 Macd4H")
	if f.Template != SkillGenerateStrategyCognition {
		t.Fatalf("template=%s", f.Template)
	}
}

func TestNewStrategyDevFlowTemplate(t *testing.T) {
	f := newStrategyDevFlow("策略开发 Macd4H")
	if f.Template != SkillStrategyDev {
		t.Fatalf("template=%s", f.Template)
	}
	if f.Phase != PhaseDevPick {
		t.Fatalf("phase=%s", f.Phase)
	}
}
