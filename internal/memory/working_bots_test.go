package memory

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestWorkingStoreApplyReportBotCodesTypedSlice(t *testing.T) {
	store := NewWorkingStore(infra.NewStateStore(t.TempDir()))
	w, err := store.Create("run-bots", "stock_premarket")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.Apply(w, "get_report_bot_codes", tools.Result{
		Status: tools.StatusOK,
		Data: map[string]any{
			"bots": []map[string]any{{
				"code": "SPCX.US", "stock_name": "SpaceX",
				"bot_id": "abc", "bot_name": "SpaceX-DCA", "bot_type": "DCA",
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.BotCodes) != 1 || updated.BotCodes[0].Code != "SPCX.US" {
		t.Fatalf("bot_codes=%+v", updated.BotCodes)
	}
	if updated.Stocks["SPCX.US"].Status != "pending" {
		t.Fatalf("stock=%+v", updated.Stocks["SPCX.US"])
	}
}
