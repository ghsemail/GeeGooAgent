package cognition

import "testing"

func TestPreferAmbiguousClarifyBareIndicator(t *testing.T) {
	msg := "这个MACD信号平时该怎么用比较好"
	if !PreferAmbiguousClarify(msg) {
		t.Fatalf("expected bare indicator usage to prefer clarify")
	}
}

func TestPreferAmbiguousClarifyCompound(t *testing.T) {
	msg := "帮我把中际旭创分析一下，然后再跑个回测看看效果"
	if !PreferAmbiguousClarify(msg) {
		t.Fatalf("expected compound analyze+backtest to prefer clarify")
	}
}

func TestPreferAmbiguousClarifyStockQuote(t *testing.T) {
	if !PreferAmbiguousClarify("腾讯股价怎么样") {
		t.Fatal("expected ambiguous stock quote")
	}
	if PreferAmbiguousClarify("帮我查询下腾讯股价") {
		t.Fatal("explicit quote request should not prefer clarify")
	}
	if PreferAmbiguousClarify("帮我分析下腾讯最近一个月的价格走势") {
		t.Fatal("explicit analysis request should not prefer clarify")
	}
}

func TestPreferAmbiguousClarifyExplicitBacktest(t *testing.T) {
	if PreferAmbiguousClarify("用MACD跑个回测看看") {
		t.Fatal("explicit backtest should not prefer clarify")
	}
}
