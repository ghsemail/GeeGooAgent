package domaincatalog_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/domaincatalog"
)

func TestVerifyExecutionProfilePriceSnapshot(t *testing.T) {
	ok, detail := domaincatalog.VerifyExecutionProfile(
		domaincatalog.ProfileStockPriceSnapshot,
		[]string{"search_code", "get_current_price"},
		[]string{"search_code"},
	)
	if !ok {
		t.Fatalf("expected snapshot pass, got %s", detail)
	}

	ok, _ = domaincatalog.VerifyExecutionProfile(
		domaincatalog.ProfileStockPriceSnapshot,
		[]string{"search_code", "get_mcp_analysis"},
		[]string{"search_code"},
	)
	if ok {
		t.Fatal("snapshot profile should require get_current_price")
	}
}

func TestVerifyExecutionProfilePriceViaMCP(t *testing.T) {
	ok, detail := domaincatalog.VerifyExecutionProfile(
		domaincatalog.ProfileStockPriceViaMCP,
		[]string{"search_code", "get_current_price"},
		[]string{"search_code", "get_current_price"},
	)
	if ok {
		t.Fatalf("expected price shortcut fail, got %s", detail)
	}

	ok, _ = domaincatalog.VerifyExecutionProfile(
		domaincatalog.ProfileStockPriceViaMCP,
		[]string{"search_code", "get_mcp_analysis"},
		[]string{"search_code", "get_mcp_analysis"},
	)
	if !ok {
		t.Fatal("expected pass for mcp path")
	}
}

func TestVerifyExecutionProfileTechnicalFullAllowsPriorSearchCode(t *testing.T) {
	ok, detail := domaincatalog.VerifyExecutionProfile(
		domaincatalog.ProfileStockTechnicalFull,
		[]string{"get_single_prompt_template", "get_mcp_analysis"},
		[]string{"search_code", "get_current_price", "get_single_prompt_template", "get_mcp_analysis"},
	)
	if !ok {
		t.Fatalf("expected pass when search_code in session, got %s", detail)
	}
}

func TestVerifyExecutionProfileContextFollowup(t *testing.T) {
	ok, detail := domaincatalog.VerifyExecutionProfile(
		domaincatalog.ProfileStockContextFollowup,
		[]string{"get_mcp_analysis", "get_current_price"},
		[]string{"search_code", "get_mcp_analysis"},
	)
	if !ok {
		t.Fatalf("expected pass, got %s", detail)
	}

	ok, _ = domaincatalog.VerifyExecutionProfile(
		domaincatalog.ProfileStockContextFollowup,
		[]string{"get_mcp_analysis"},
		[]string{"get_mcp_analysis"},
	)
	if ok {
		t.Fatal("expected fail without session search_code")
	}
}

func TestVerifyExecutionProfileSymbolResolve(t *testing.T) {
	ok, _ := domaincatalog.VerifyExecutionProfile(
		domaincatalog.ProfileStockSymbolResolve,
		[]string{"search_code", "get_mcp_analysis"},
		[]string{"search_code", "get_mcp_analysis"},
	)
	if !ok {
		t.Fatal("expected pass when judged turn searches code")
	}

	ok, _ = domaincatalog.VerifyExecutionProfile(
		domaincatalog.ProfileStockSymbolResolve,
		[]string{"get_mcp_analysis"},
		[]string{"search_code", "get_mcp_analysis"},
	)
	if ok {
		t.Fatal("expected fail when search_code only in prior turn")
	}
}

func TestExecutionProfileForQuotePrice(t *testing.T) {
	if domaincatalog.ExecutionProfileFor(domaincatalog.DomainStockAnalysis, domaincatalog.StockActQuotePrice) != domaincatalog.ProfileStockPriceSnapshot {
		t.Fatal("quote_price should map to price_snapshot")
	}
}

func TestVerifyExecutionProfileSubagentMultiStock(t *testing.T) {
	ok, detail := domaincatalog.VerifyExecutionProfile(
		domaincatalog.ProfileSubagentMultiStock,
		[]string{"delegate_tasks"},
		nil,
	)
	if !ok {
		t.Fatalf("expected delegate_tasks pass, got %s", detail)
	}

	ok, _ = domaincatalog.VerifyExecutionProfile(
		domaincatalog.ProfileSubagentMultiStock,
		[]string{"search_code", "get_mcp_analysis"},
		nil,
	)
	if ok {
		t.Fatal("main agent should not analyze stocks directly")
	}

	ok, _ = domaincatalog.VerifyExecutionProfile(
		domaincatalog.ProfileSubagentMultiStock,
		[]string{"delegate_tasks", "search_code"},
		nil,
	)
	if ok {
		t.Fatal("forbid search_code on judged turn when delegating")
	}
}

func TestExecutionProfileForMultiSymbolDelegate(t *testing.T) {
	got := domaincatalog.ExecutionProfileFor(domaincatalog.DomainStockAnalysis, domaincatalog.StockActMultiSymbol)
	if got != domaincatalog.ProfileSubagentMultiStock {
		t.Fatalf("multi_symbol_delegate profile = %q want %s", got, domaincatalog.ProfileSubagentMultiStock)
	}
}

func TestProfileExecutionHintFromRequiredTools(t *testing.T) {
	hint := domaincatalog.ProfileExecutionHint(domaincatalog.ProfileSignalProbeExecute)
	if hint != "本轮须调用: probe_bot_signal_series" {
		t.Fatalf("hint=%q", hint)
	}
	hint = domaincatalog.ProfileExecutionHint(domaincatalog.ProfileSignalProbeSymbolSwitch)
	if hint != "本轮须调用: search_code, probe_bot_signal_series" {
		t.Fatalf("hint=%q", hint)
	}
}

func TestExecutionProfileForSignalProbeExecute(t *testing.T) {
	got := domaincatalog.ProbeExecutionProfile(domaincatalog.DomainSignalProbe, "probe", "换一个策略")
	if got != domaincatalog.ProfileSignalProbeExecute {
		t.Fatalf("strategy swap profile = %q want %s", got, domaincatalog.ProfileSignalProbeExecute)
	}
	got = domaincatalog.ProbeExecutionProfile(domaincatalog.DomainSignalProbe, "probe", "00700")
	if got != domaincatalog.ProfileSignalProbeSymbolSwitch {
		t.Fatalf("symbol switch profile = %q want %s", got, domaincatalog.ProfileSignalProbeSymbolSwitch)
	}
	ok, detail := domaincatalog.VerifyExecutionProfile(
		domaincatalog.ProfileSignalProbeExecute,
		[]string{"clarify", "get_signal_combinations"},
		nil,
	)
	if ok {
		t.Fatalf("clarify-only turn must miss probe: %s", detail)
	}
	ok, _ = domaincatalog.VerifyExecutionProfile(
		domaincatalog.ProfileSignalProbeExecute,
		[]string{"clarify", "probe_bot_signal_series"},
		nil,
	)
	if !ok {
		t.Fatal("clarify + probe must satisfy signal_probe profile")
	}
}
