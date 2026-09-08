package domaincatalog_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/domaincatalog"
)

func TestNormalizeStockAct(t *testing.T) {
	if domaincatalog.NormalizeStockAct("quote_price") != domaincatalog.StockActQuotePrice {
		t.Fatal("quote_price")
	}
	if domaincatalog.NormalizeStockAct("bogus") != domaincatalog.StockActAnalyze {
		t.Fatal("unknown -> analyze")
	}
}

func TestExecutionProfileForStockActs(t *testing.T) {
	if domaincatalog.ExecutionProfileFor(domaincatalog.DomainStockAnalysis, domaincatalog.StockActQuotePrice) != domaincatalog.ProfileStockPriceSnapshot {
		t.Fatal("quote_price profile")
	}
	if domaincatalog.ExecutionProfileFor(domaincatalog.DomainChat, domaincatalog.StockActQuotePrice) != "" {
		t.Fatal("non-stock domain should have no profile")
	}
}
