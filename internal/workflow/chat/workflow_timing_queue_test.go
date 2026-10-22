package chat

import (
	"testing"
	"time"
)

func TestWorkflowTotalIncludesQueueWait(t *testing.T) {
	flow := &Flow{WorkflowStartedAt: time.Now().Add(-100 * time.Millisecond)}
	flow.PhaseTimingsMs = map[string]int64{"_queue_wait": 61000}
	recordWorkflowTimingTotal(flow)
	if flow.PhaseTimingsMs["_total"] < 61000 {
		t.Fatalf("total=%d", flow.PhaseTimingsMs["_total"])
	}
}
