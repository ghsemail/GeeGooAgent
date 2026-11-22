package diagnoseskill_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/diagnoseskill"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func poolJEVSrv(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/decision/choice":
			_, _ = w.Write([]byte(`{"code":0,"data":{"model":"m","answers":{"direction":{"choice":"看多","probabilities":{"看多":0.75,"看空":0.15,"不明确":0.1},"confidence":0.75}}}}`))
		case "/v1/decision/chat":
			_, _ = w.Write([]byte(`{"code":0,"data":{"content":"## 整体结论\n池级看多，置信 0.75。\n1. BTC 强\n2. ETH 跟进"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func testPoolInput(srvURL string) diagnoseskill.MacroPoolRunInput {
	return diagnoseskill.MacroPoolRunInput{
		Date: "2026-11-20", PoolWorkflow: "crypto", SourceBatchID: "batch-1", Title: "加密",
		ExpectedSymbols: []string{"BTC-USD.CRYPTO", "ETH-USD.CRYPTO"},
		MinReasonLen:    20,
		Symbols: []diagnoseskill.MacroPoolSymbolRow{
			{Symbol: "BTC-USD.CRYPTO", ResultID: "sym:btc", JEVDirection: "long", JEVReason: "BTC 偏多测试理由足够长度", Confidence: 0.8},
			{Symbol: "ETH-USD.CRYPTO", ResultID: "sym:eth", JEVDirection: "long", JEVReason: "ETH 偏多测试理由足够长度", Confidence: 0.7},
		},
		JEV: strategydev.NewJevClient(strategydev.JevConfig{BaseURL: srvURL, APIKey: "k"}),
	}
}

func TestValidateMacroPoolRejectsDuplicateSymbol(t *testing.T) {
	in := testPoolInput("")
	in.Symbols = append(in.Symbols, in.Symbols[0])
	if err := diagnoseskill.ValidateMacroPoolInput(in); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateMacroPoolRejectsMissingExpectedSymbol(t *testing.T) {
	in := testPoolInput("")
	in.Symbols = in.Symbols[:1]
	err := diagnoseskill.ValidateMacroPoolInput(in)
	if err == nil || (!strings.Contains(err.Error(), "missing expected") && !strings.Contains(err.Error(), "symbols count")) {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateMacroPoolRejectsUnexpectedExtraSymbol(t *testing.T) {
	in := testPoolInput("")
	in.Symbols = append(in.Symbols, diagnoseskill.MacroPoolSymbolRow{
		Symbol: "SOL-USD.CRYPTO", ResultID: "sym:sol", JEVDirection: "long",
		JEVReason: "SOL extra symbol reason long enough",
	})
	err := diagnoseskill.ValidateMacroPoolInput(in)
	if err == nil || (!strings.Contains(err.Error(), "unexpected symbol") && !strings.Contains(err.Error(), "symbols count")) {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateMacroPoolRejectsNoExpectedSymbols(t *testing.T) {
	in := testPoolInput("")
	in.ExpectedSymbols = nil
	if err := diagnoseskill.ValidateMacroPoolInput(in); err == nil {
		t.Fatal("expected error")
	}
}

func TestRunPoolSuccessShape(t *testing.T) {
	srv := poolJEVSrv(t)
	defer srv.Close()
	res, err := diagnoseskill.RunPool(context.Background(), testPoolInput(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if res["jev_direction"] != "long" {
		t.Fatalf("dir=%v", res["jev_direction"])
	}
	if res["detail_ref"] != "diagnose_macro_pool" {
		t.Fatal(res["detail_ref"])
	}
	if _, has := res["jev_unified"]; has {
		t.Fatal("unexpected jev_unified")
	}
	if _, has := res["symbols"]; has {
		t.Fatal("symbols echo should be removed; use input_refs")
	}
	refsRaw, ok := res["input_refs"]
	if !ok {
		t.Fatal("missing input_refs")
	}
	refsJSON := strings.TrimSpace(fmt.Sprint(refsRaw))
	if !strings.Contains(refsJSON, "sym:btc") || !strings.Contains(refsJSON, "sym:eth") {
		t.Fatalf("input_refs=%v", refsRaw)
	}
}

func TestRunPoolNeutralSuccessAllowed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/decision/choice" {
			_, _ = w.Write([]byte(`{"code":0,"data":{"model":"m","answers":{"direction":{"choice":"不明确","probabilities":{"看多":0.3,"看空":0.3,"不明确":0.4},"confidence":0.4}}}}`))
			return
		}
		_, _ = w.Write([]byte(`{"code":0,"data":{"content":"## 整体结论\n不明确，双方摘要均未形成一致方向。"}}`))
	}))
	defer srv.Close()
	in := testPoolInput(srv.URL)
	in.Symbols[0].JEVDirection = "long"
	in.Symbols[1].JEVDirection = "short"
	res, err := diagnoseskill.RunPool(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if res["jev_direction"] != "neutral" {
		t.Fatalf("dir=%v", res["jev_direction"])
	}
	if !res["conflict"].(bool) {
		t.Fatal("expected conflict true for long+short inputs even when choice unclear")
	}
}

func TestRunPoolChoiceFailureNoNeutralFake(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "jev down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	_, err := diagnoseskill.RunPool(context.Background(), testPoolInput(srv.URL))
	if err == nil {
		t.Fatal("expected error")
	}
}