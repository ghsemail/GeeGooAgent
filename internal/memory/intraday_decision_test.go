package memory

import "testing"

func TestDecideIntradayBuyAligned(t *testing.T) {
	ws := StockWorkspace{
		TradeType: "信号买入", PreMarketResult: "long", PreMarketConfidence: "high",
		CurrentPrice: 100,
	}
	result, confidence := DecideIntraday(ws)
	if result != "buy" {
		t.Fatalf("expected buy, got %s", result)
	}
	if confidence != "medium" {
		t.Fatalf("expected medium confidence, got %s", confidence)
	}
}

func TestDecideIntradayBuyBlockedByPreMarketShort(t *testing.T) {
	ws := StockWorkspace{
		TradeType: "信号买入", PreMarketResult: "short", PreMarketConfidence: "high",
	}
	result, _ := DecideIntraday(ws)
	if result != "hold" {
		t.Fatalf("expected hold, got %s", result)
	}
}

func TestDecideIntradaySellWithoutPosition(t *testing.T) {
	ws := StockWorkspace{TradeType: "信号卖出", BotType: "DCA", HasPosition: false}
	result, _ := DecideIntraday(ws)
	if result != "hold" {
		t.Fatalf("expected hold, got %s", result)
	}
}

func TestFinalizeDerivedFieldsIntradayStock(t *testing.T) {
	w := NewPreMarketWorking("s1", "intraday_stock")
	w.Stocks["00700.HK"] = StockWorkspace{
		Code: "00700.HK", TradeType: "信号买入", PreMarketResult: "long",
		CurrentPrice: 500,
	}
	ws := w.Stocks["00700.HK"]
	finalizeDerivedFields(w, &ws, "00700.HK")
	if ws.IntradayResult != "buy" {
		t.Fatalf("expected buy, got %s", ws.IntradayResult)
	}
}
