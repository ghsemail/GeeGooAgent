package verify_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/verify"
)

func goodReport() map[string]any {
	long := strings.Repeat("引用 [ev_1] 资金流 +3.5亿; ", 12) // >80 chars
	return map[string]any{
		"code": "00700.HK", "stock_name": "腾讯控股", "report_id": "r1",
		"bot_id": "b1", "bot_name": "bot", "bot_type": "DCA",
		"result": "long", "confidence": "medium", "suggestion": "hold",
		"reason": long, "report": "# Pre-market", "evidence_refs": []any{"ev_1"},
	}
}

func TestVerifyReportPassesOnCompleteRecord(t *testing.T) {
	t.Parallel()
	card := verify.VerifyReport(goodReport())
	if !card.Passed {
		t.Fatalf("expected pass: %s", card.Summary())
	}
}

func TestVerifyReportFailsOnMissingBotID(t *testing.T) {
	t.Parallel()
	r := goodReport()
	delete(r, "bot_id")
	card := verify.VerifyReport(r)
	if card.Passed {
		t.Fatal("expected fail when bot_id missing")
	}
}

func TestVerifyReportFailsOnInvalidResultEnum(t *testing.T) {
	t.Parallel()
	r := goodReport()
	r["result"] = "maybe"
	card := verify.VerifyReport(r)
	if card.Passed {
		t.Fatal("expected fail on invalid result enum")
	}
}

func TestVerifyReportFailsOnShortReason(t *testing.T) {
	t.Parallel()
	r := goodReport()
	r["reason"] = "太短"
	card := verify.VerifyReport(r)
	if card.Passed {
		t.Fatal("expected fail on short reason")
	}
}

func TestVerifyReportFailsOnEmptyEvidenceRefs(t *testing.T) {
	t.Parallel()
	r := goodReport()
	r["evidence_refs"] = []any{}
	card := verify.VerifyReport(r)
	if card.Passed {
		t.Fatal("expected fail on empty evidence_refs")
	}
}

func TestAllPassAndCompletenessMatrix(t *testing.T) {
	t.Parallel()
	good := goodReport()
	bad := goodReport()
	delete(bad, "bot_id")
	cards := verify.VerifyReports([]map[string]any{good, bad})
	if verify.AllPass(cards) {
		t.Fatal("AllPass should be false with one bad card")
	}
	matrix := verify.CompletenessMatrix(cards)
	if matrix["bot_id"] != 0.5 {
		t.Fatalf("bot_id completeness = %v, want 0.5", matrix["bot_id"])
	}
}
