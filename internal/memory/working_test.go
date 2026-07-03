package memory

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestWorkingStoreApplyCapturesEvidenceRefs(t *testing.T) {
	store := NewWorkingStore(infra.NewStateStore(t.TempDir()))
	w, err := store.Create("run-test", "pre_market")
	if err != nil {
		t.Fatal(err)
	}
	w.Stocks["00700.HK"] = StockWorkspace{Code: "00700.HK", Status: "pending"}
	if err := store.Save(w); err != nil {
		t.Fatal(err)
	}

	updated, err := store.Apply(w, "get_mcp_analysis", tools.Result{
		Status:  tools.StatusOK,
		Summary: "weekly analysis",
		Data: map[string]any{
			"code": "00700.HK", "period": "weekly", "analysis_result": "weekly trend is neutral",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.EvidenceRefs) != 1 {
		t.Fatalf("evidence refs=%d", len(updated.EvidenceRefs))
	}
	ref := updated.EvidenceRefs[0]
	if !strings.HasPrefix(ref.ID, "ev_") {
		t.Fatalf("unexpected evidence id: %s", ref.ID)
	}
	if ref.RunID != "run-test" || ref.Tool != "get_mcp_analysis" {
		t.Fatalf("unexpected evidence identity: %+v", ref)
	}
	if ref.Source != "stock.00700.HK.weekly_analysis" {
		t.Fatalf("source=%s", ref.Source)
	}
	if ref.PayloadHash == "" || ref.ObservedAt == "" || ref.Summary == "" {
		t.Fatalf("incomplete evidence ref: %+v", ref)
	}

	loaded, err := store.Load("run-test")
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.EvidenceRefs) != 1 || loaded.EvidenceRefs[0].ID != ref.ID {
		t.Fatalf("evidence round-trip failed: %+v", loaded.EvidenceRefs)
	}
}
