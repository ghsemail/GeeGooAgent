package chat

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func TestIsStrategyDevPipelineIntent(t *testing.T) {
	if !IsStrategyDevPipelineIntent("策略开发 00700.HK 60分钟", nil) {
		t.Fatal("expected pipeline with symbol and frequency")
	}
	if IsStrategyDevPipelineIntent("策略开发 Macd4H", nil) {
		t.Fatal("strategy name only should be archive flow")
	}
	if IsStrategyDevPipelineIntent("读取 Macd4H 策略", nil) {
		t.Fatal("read archive should not be pipeline")
	}
	opts := &runtime.StrategyDevOpts{Code: "00700.HK", Frequency: "60m"}
	if !IsStrategyDevPipelineIntent("策略开发", opts) {
		t.Fatal("expected pipeline from workflow_options")
	}
}

func TestShouldStartStrategyDevPipelineFlow(t *testing.T) {
	if !ShouldStartStrategyDevPipelineFlow("策略开发 00700.HK 60m", nil, nil) {
		t.Fatal("expected new pipeline flow")
	}
	if ShouldStartStrategyDevPipelineFlow("策略开发 共振", nil, nil) {
		t.Fatal("strategy name should not start pipeline")
	}
	active := &Flow{Template: SkillMultiStrategyCompare, Status: StatusRunning}
	if ShouldStartStrategyDevPipelineFlow("策略开发 00700.HK 60m", nil, active) {
		t.Fatal("should not start when another flow is active")
	}
}

func TestNewStrategyDevPipelineFlow(t *testing.T) {
	f := newStrategyDevPipelineFlow("策略开发 00700.HK 60分钟", nil)
	if f.Template != SkillStrategyDevPipeline {
		t.Fatalf("template=%s", f.Template)
	}
	if f.StockCode == "" || f.ProbeFrequencyOverride == "" {
		t.Fatalf("code=%q freq=%q", f.StockCode, f.ProbeFrequencyOverride)
	}
}
