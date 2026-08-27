package tools

import (
	"context"
	"testing"
)

func TestAutoClarifyChoicePrefersSignalAnd60m(t *testing.T) {
	if got, ok := AutoClarifyChoice("用哪一条 SAR 抛物线？", []string{
		"SAR抛物线信号（buy/sell 触发型，适合跑回测）",
		"SAR抛物线趋势（趋势方向型，flag 类）",
	}); !ok || got != "SAR抛物线信号（buy/sell 触发型，适合跑回测）" {
		t.Fatalf("got=%q ok=%v", got, ok)
	}
	if got, ok := AutoClarifyChoice("回测频率用哪个？", []string{
		"daily（日线，长期）",
		"60m（小时，中短线）",
		"5m（5 分钟，短线）",
	}); !ok || got != "60m（小时，中短线）" {
		t.Fatalf("got=%q ok=%v", got, ok)
	}
}

func TestHandleClarifyAutoPicksWhenUnanswered(t *testing.T) {
	res := handleClarify(Context{
		ClarifyFn: func(context.Context, string, []string) (string, bool) { return "", false },
	}, map[string]any{
		"question": "回测频率用哪个？",
		"choices":  []any{"daily（日线，长期）", "60m（小时，中短线）"},
	})
	if res.Status != StatusOK || res.Data["user_response"] != "60m（小时，中短线）" {
		t.Fatalf("res=%+v", res)
	}
}
