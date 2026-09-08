package domaincatalog_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/domaincatalog"
)

func TestRefineStockAnalysisAct(t *testing.T) {
	cases := []struct {
		msg  string
		last domaincatalog.Domain
		want string
	}{
		{msg: "帮我查一下腾讯控股现在的股价", want: domaincatalog.StockActQuotePrice},
		{msg: "再帮我看看腾讯的技术面和K线图", last: domaincatalog.DomainStockAnalysis, want: domaincatalog.StockActTechnicalAnalysis},
		{msg: "它最近走势怎么样", last: domaincatalog.DomainStockAnalysis, want: domaincatalog.StockActContextFollowup},
		{msg: "不聊中际旭创了，帮我分析一下贵州茅台", last: domaincatalog.DomainStockAnalysis, want: domaincatalog.StockActSymbolResolve},
		{msg: "帮我分析一下中际旭创", want: domaincatalog.StockActAnalyze},
	}
	for _, tc := range cases {
		got := domaincatalog.RefineStockAnalysisAct(tc.msg, tc.last)
		if got != tc.want {
			t.Fatalf("%q last=%s: got %s want %s", tc.msg, tc.last, got, tc.want)
		}
	}
}

func TestExecutionProfileForStockActs(t *testing.T) {
	if domaincatalog.ExecutionProfileFor(domaincatalog.DomainStockAnalysis, domaincatalog.StockActQuotePrice) != domaincatalog.ProfileStockPriceViaMCP {
		t.Fatal("quote_price profile")
	}
	if domaincatalog.ExecutionProfileFor(domaincatalog.DomainChat, domaincatalog.StockActQuotePrice) != "" {
		t.Fatal("non-stock domain should have no profile")
	}
}
