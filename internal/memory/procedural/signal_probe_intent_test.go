package procedural

import "testing"

func TestSignalProbeIntent(t *testing.T) {
	msg := "请帮我用「SAR抛物线」策略测试一下腾讯控股（0700.HK · 港股）。"
	if !SignalProbeIntent(msg) {
		t.Fatal("eval signal probe message should count as signal probe intent")
	}
	if SignalProbeIntent("帮我回测一下 sar+macd 在小米") {
		t.Fatal("explicit backtest should not count as signal probe")
	}
	if !SignalProbeIntent("帮我测一下有没有买卖信号") {
		t.Fatal("expected signal-only probe intent")
	}
}
