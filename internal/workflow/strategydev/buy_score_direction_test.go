package strategydev

import "testing"

func TestBuyScoreDirectionSoftNormalAndReverse(t *testing.T) {
	const th = 0.0
	if !WouldInterceptTriggerByScoreMode("buy", -10, th, BuyScoreDirectionNormal, true) {
		t.Fatal("normal: negative buy should would_intercept")
	}
	if WouldInterceptTriggerByScoreMode("buy", 10, th, BuyScoreDirectionNormal, true) {
		t.Fatal("normal: positive buy should not would_intercept")
	}
	if !WouldInterceptTriggerByScoreMode("buy", 10, th, BuyScoreDirectionReverse, true) {
		t.Fatal("reverse: positive buy should would_intercept")
	}
	if WouldInterceptTriggerByScoreMode("buy", -10, th, BuyScoreDirectionReverse, true) {
		t.Fatal("reverse: negative buy should not would_intercept")
	}
	if WouldInterceptTriggerByScoreMode("buy", 0, th, BuyScoreDirectionReverse, true) {
		t.Fatal("reverse: zero score should not would_intercept")
	}
}

func TestSoftSellNeverWouldInterceptByScore(t *testing.T) {
	if WouldInterceptTriggerByScoreMode("sell", 50, 0, BuyScoreDirectionNormal, true) {
		t.Fatal("soft sell must not would_intercept from score")
	}
	if WouldInterceptTriggerByScoreMode("sell", -50, 0, BuyScoreDirectionReverse, true) {
		t.Fatal("soft sell must not would_intercept in reverse mode either")
	}
}

func TestHardGateSellScoreUnchanged(t *testing.T) {
	if !WouldInterceptTriggerByScoreMode("sell", 25, 0, BuyScoreDirectionReverse, false) {
		t.Fatal("hard gate sell should still intercept on bullish score")
	}
}
