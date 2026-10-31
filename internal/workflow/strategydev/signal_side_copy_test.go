package strategydev

import (
	"strings"
	"testing"
)

func TestLabSignalSideTriggerHeadingLongOnly(t *testing.T) {
	if got := LabSignalSideTriggerHeading("sell", true); !strings.Contains(got, "平多") {
		t.Fatalf("sell long_only=%q", got)
	}
	if got := LabSignalSideTriggerHeading("sell", false); got != "信号侧：卖出触发" {
		t.Fatalf("sell both=%q", got)
	}
}

func TestSignalDiagnoseProbeHitLineLongOnly(t *testing.T) {
	line := SignalDiagnoseProbeHitLine(true, 16, 21)
	if strings.Contains(line, "买入 16 次 / 卖出") {
		t.Fatalf("must not use bare 卖出 line: %q", line)
	}
	if !strings.Contains(line, "只做多") || !strings.Contains(line, "平多") {
		t.Fatalf("line=%q", line)
	}
}
