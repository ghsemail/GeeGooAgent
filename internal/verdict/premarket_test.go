package verdict_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/verdict"
)

func TestArbitrateStockAligned(t *testing.T) {
	t.Parallel()
	v := verdict.ArbitrateStockPreMarket(verdict.StockPreMarketInput{
		Attitude: "bullish", EvidenceCount: 6, HasWeekly: true, HasCapitalFlow: true,
		SuggestedResult: "long", SuggestedConfidence: "high",
		MarketResult: "long",
	})
	if v.Result != "long" || v.Confidence != "high" {
		t.Fatalf("got %+v", v)
	}
}

func TestArbitrateStockConflictKeepsAttitude(t *testing.T) {
	t.Parallel()
	v := verdict.ArbitrateStockPreMarket(verdict.StockPreMarketInput{
		Attitude: "bullish", EvidenceCount: 4, HasWeekly: true,
		SuggestedResult: "short", SuggestedConfidence: "high",
	})
	if v.Result != "long" {
		t.Fatalf("result=%s", v.Result)
	}
	if v.Note == "" {
		t.Fatal("expected arbitration note")
	}
}

func TestArbitrateStockTripleConflictNeutral(t *testing.T) {
	t.Parallel()
	v := verdict.ArbitrateStockPreMarket(verdict.StockPreMarketInput{
		Attitude: "bullish", EvidenceCount: 5, HasWeekly: true,
		SuggestedResult: "short", SuggestedConfidence: "medium",
		MarketResult: "short",
	})
	if v.Result != "neutral" || v.Confidence != verdict.ConfidenceReview {
		t.Fatalf("got %+v", v)
	}
}

func TestArbitrateStockUpgradeFromNeutral(t *testing.T) {
	t.Parallel()
	v := verdict.ArbitrateStockPreMarket(verdict.StockPreMarketInput{
		Attitude: "", EvidenceCount: 5, HasWeekly: true, HasCapitalFlow: true, CapitalRequired: true,
		SuggestedResult: "long", SuggestedConfidence: "medium",
		MarketResult: "long",
	})
	if v.Result != "long" {
		t.Fatalf("result=%s", v.Result)
	}
}

func TestArbitrateMarketUpgrade(t *testing.T) {
	t.Parallel()
	v := verdict.ArbitrateMarketPreMarket(verdict.MarketPreMarketInput{
		IndicesDone: true, MarketNewsDone: true, EvidenceCount: 3,
		SuggestedResult: "long", SuggestedConfidence: "high",
	})
	if v.Result != "long" || v.Confidence != "high" {
		t.Fatalf("got %+v", v)
	}
}

func TestArbitrateMarketIncompleteStaysNeutral(t *testing.T) {
	t.Parallel()
	v := verdict.ArbitrateMarketPreMarket(verdict.MarketPreMarketInput{
		IndicesDone: true, MarketNewsDone: false,
		SuggestedResult: "long", SuggestedConfidence: "high",
	})
	if v.Result != "neutral" {
		t.Fatalf("result=%s", v.Result)
	}
}
