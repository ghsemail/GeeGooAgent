package memory

import "testing"

func TestDecideIntradayBuyAligned(t *testing.T) {
	ws := StockWorkspace{
		TradeType: "信号买入", PreMarketResult: "long", PreMarketConfidence: "high",
		AttitudeSwitch: true, CurrentPrice: 100,
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
		AttitudeSwitch: true,
	}
	result, _ := DecideIntraday(ws)
	if result != "hold" {
		t.Fatalf("expected hold, got %s", result)
	}
}

func TestDecideIntradayBuyBlockedByHourlyBearish(t *testing.T) {
	ws := StockWorkspace{
		TradeType: "信号买入", PreMarketResult: "long", PreMarketConfidence: "high",
		AttitudeSwitch: true,
		HourlySignalAnalysis: "> **整体结论**：小时级信号偏空，短线承压。",
		CurrentPrice:         100,
	}
	result, _ := DecideIntraday(ws)
	if result != "hold" {
		t.Fatalf("expected hold, got %s", result)
	}
}

func TestDecideIntradaySellBlockedByHourlyBullish(t *testing.T) {
	ws := StockWorkspace{
		TradeType: "信号卖出", BotType: "DCA", HasPosition: true,
		AttitudeSwitch: true,
		PreMarketResult: "short", PreMarketConfidence: "medium",
		HourlyPriceAnalysis: "> **整体判断**：价格结构偏多，反弹动能增强。",
		CurrentPrice:        100,
	}
	result, _ := DecideIntraday(ws)
	if result != "hold" {
		t.Fatalf("expected hold, got %s", result)
	}
}

func TestDecideIntradayBuyAllowedWithoutHourly(t *testing.T) {
	ws := StockWorkspace{
		TradeType: "信号买入", PreMarketResult: "long", AttitudeSwitch: true, CurrentPrice: 100,
	}
	result, _ := DecideIntraday(ws)
	if result != "buy" {
		t.Fatalf("expected buy, got %s", result)
	}
}

func TestDecideIntradaySellWithoutPosition(t *testing.T) {
	ws := StockWorkspace{TradeType: "信号卖出", BotType: "DCA", HasPosition: false}
	result, _ := DecideIntraday(ws)
	if result != "hold" {
		t.Fatalf("expected hold, got %s", result)
	}
}

func TestDecideIntradayBuyWithoutPremarketWhenAttitudeOn(t *testing.T) {
	ws := StockWorkspace{
		TradeType: "信号买入", AttitudeSwitch: true, CurrentPrice: 100,
		HourlyPriceAnalysis: "偏多",
	}
	result, confidence := DecideIntraday(ws)
	if result != "hold" {
		t.Fatalf("expected hold without premarket, got %s", result)
	}
	if confidence != "low" {
		t.Fatalf("expected low confidence, got %s", confidence)
	}
}

func TestDecideIntradayBuyWithoutPremarketWhenAttitudeOff(t *testing.T) {
	ws := StockWorkspace{
		TradeType: "信号买入", AttitudeSwitch: false, CurrentPrice: 100,
		HourlyPriceAnalysis: "偏多",
	}
	result, _ := DecideIntraday(ws)
	if result != "buy" {
		t.Fatalf("expected buy when premarket skipped, got %s", result)
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
	if ws.IntradayResult != "" {
		t.Fatalf("expected deferred decision, got %s", ws.IntradayResult)
	}
}
