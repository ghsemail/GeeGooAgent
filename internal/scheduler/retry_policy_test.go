package scheduler_test

import (
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/scheduler"
)

func TestJobRetryPolicyDefaults(t *testing.T) {
	t.Parallel()
	interval, max := scheduler.JobRetryPolicy()
	if interval != 5*time.Minute {
		t.Fatalf("interval=%s", interval)
	}
	if max != 3 {
		t.Fatalf("max=%d", max)
	}
}
