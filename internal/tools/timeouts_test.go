package tools

import (
	"testing"
	"time"
)

func TestExecutionTimeout(t *testing.T) {
	t.Parallel()
	defaultTO := 120 * time.Second
	if got := ExecutionTimeout("search_code", defaultTO); got != defaultTO {
		t.Fatalf("search_code timeout = %v, want default %v", got, defaultTO)
	}
	if got := ExecutionTimeout("generate_dca_strategy", defaultTO); got != 4*60*time.Second {
		t.Fatalf("generate_dca_strategy timeout = %v", got)
	}
	if got := ExecutionTimeout("clarify", defaultTO); got != 10*time.Minute {
		t.Fatalf("clarify timeout = %v", got)
	}
}
