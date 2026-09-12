package agent

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestDelegateStepExtra(t *testing.T) {
	extra := delegateStepExtra("delegate_tasks", tools.Result{
		Data: map[string]any{
			"results": []map[string]any{{"task": "a", "status": "ok"}},
			"ok":      1,
			"failed":  0,
		},
	})
	if extra == nil || extra["results"] == nil {
		t.Fatalf("expected results in extra: %#v", extra)
	}
	if delegateStepExtra("search_code", tools.Result{Data: map[string]any{"x": 1}}) != nil {
		t.Fatal("non-delegate tool should not get extra")
	}
}
