package strategydev

import (
	"testing"
	"time"
)

func TestDefaultLabDetailJEVTimeoutBudget(t *testing.T) {
	t.Helper()
	if DefaultLabDetailJEVTimeout < 3*time.Minute {
		t.Fatalf("JEV phase budget too small for four lanes + unified: %v", DefaultLabDetailJEVTimeout)
	}
	if DefaultJevHTTPRequestTimeout > DefaultLabDetailJEVTimeout {
		t.Fatalf("per Decision HTTP timeout %v exceeds JEV phase budget %v",
			DefaultJevHTTPRequestTimeout, DefaultLabDetailJEVTimeout)
	}
}