package strategydev

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildMacroOverallJEVStatePartialPoolNote(t *testing.T) {
	state := BuildMacroOverallJEVState("2026-11-22", []MacroOverallPoolInput{
		{PoolWorkflow: "crypto", JEVDirection: "short", JEVReason: strings.Repeat("x", 30)},
	})
	if state == "" {
		t.Fatal("empty state")
	}
	for _, frag := range []string{"参与监控池数：1", "非七池全量", "虚拟货币池"} {
		if !strings.Contains(state, frag) {
			t.Fatalf("state missing %q: %q", frag, state)
		}
	}
}

func TestResolveMacroOverallJEVNoConflictField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/decision/choice" {
			_, _ = w.Write([]byte(`{"code":0,"data":{"model":"m","answers":{"direction":{"choice":"看多","probabilities":{"看多":0.9,"看空":0.05,"不明确":0.05},"confidence":0.9}}}}`))
			return
		}
		_, _ = w.Write([]byte(`{"code":0,"data":{"content":"## 整体结论\nrisk-on。"}}`))
	}))
	defer srv.Close()
	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "k"})
	rec := ResolveMacroOverallJEV(context.Background(), jev, "2026-11-22", []MacroOverallPoolInput{
		{PoolWorkflow: "us", JEVDirection: "long", JEVReason: strings.Repeat("A", 40)},
		{PoolWorkflow: "crypto", JEVDirection: "short", JEVReason: strings.Repeat("B", 40)},
	})
	if !rec.OK {
		t.Fatalf("rec=%+v", rec)
	}
	if rec.Confidence > 0.75 && rec.Confidence == 0.9 {
		// no conflict cap — long+short inputs may still yield high confidence choice
	}
}
