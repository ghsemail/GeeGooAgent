package diagnoseskill_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/diagnoseskill"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func overallJEVSrv(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/decision/choice":
			_, _ = w.Write([]byte(`{"code":0,"data":{"model":"m","answers":{"direction":{"choice":"看空","probabilities":{"看多":0.2,"看空":0.65,"不明确":0.15},"confidence":0.65}}}}`))
		case "/v1/decision/chat":
			_, _ = w.Write([]byte(`{"code":0,"data":{"content":"## 整体结论\n风险偏好偏空，置信 0.65。\n1. 加密池空头\n2. 美股池观望"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

var cryptoReason = `BTC 空头信号一致性强。日线呈 LH+LL 下行结构并出现 BOS，EMA13/26 斜率、MACD 柱及 Force Index 均偏弱，ADX 与 SAR 也给出空头信号。ETH 同样由模型判定为空头占优：日线趋势汇总为空头占优（ADX、SAR 均为空，net=-2），且 EMA13/26 斜率均为负、MACD 柱与 Force Index fi13 均为负，显示下行压力。需跟踪反弹风险：BTC RSI 25.35、KD 处于低位；ETH价格结构部分分歧，RSI与Stochastic提示超卖。`

func TestValidateMacroOverallRejectsEmptyPools(t *testing.T) {
	in := diagnoseskill.MacroOverallRunInput{Date: "2026-11-22", Pools: nil}
	if err := diagnoseskill.ValidateMacroOverallInput(in); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateMacroOverallRejectsDuplicatePool(t *testing.T) {
	in := diagnoseskill.MacroOverallRunInput{
		Date: "2026-11-22",
		Pools: []diagnoseskill.MacroOverallPoolRow{
			{PoolWorkflow: "crypto", JEVDirection: "short", JEVReason: cryptoReason},
			{PoolWorkflow: "crypto", JEVDirection: "long", JEVReason: cryptoReason},
		},
	}
	if err := diagnoseskill.ValidateMacroOverallInput(in); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateMacroOverallRejectsInvalidWorkflow(t *testing.T) {
	in := diagnoseskill.MacroOverallRunInput{
		Date: "2026-11-22",
		Pools: []diagnoseskill.MacroOverallPoolRow{
			{PoolWorkflow: "overall", JEVDirection: "short", JEVReason: cryptoReason},
		},
	}
	if err := diagnoseskill.ValidateMacroOverallInput(in); err == nil {
		t.Fatal("expected error")
	}
}

func TestRunOverallSingleCryptoSuccess(t *testing.T) {
	srv := overallJEVSrv(t)
	defer srv.Close()
	res, err := diagnoseskill.RunOverall(context.Background(), diagnoseskill.MacroOverallRunInput{
		Date: "2026-11-22",
		Pools: []diagnoseskill.MacroOverallPoolRow{
			{PoolWorkflow: "crypto", JEVDirection: "short", JEVReason: cryptoReason},
		},
		JEV: strategydev.NewJevClient(strategydev.JevConfig{BaseURL: srv.URL, APIKey: "k"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res["date"] != "2026-11-22" {
		t.Fatalf("date=%v", res["date"])
	}
	if res["jev_direction"] != "short" {
		t.Fatalf("dir=%v", res["jev_direction"])
	}
	if res["detail_ref"] != "diagnose_macro_overall" {
		t.Fatal(res["detail_ref"])
	}
	if _, has := res["input_refs"]; has {
		t.Fatal("must not return input_refs")
	}
	if _, has := res["conflict"]; has {
		t.Fatal("must not return conflict")
	}
	ch, ok := res["choices"].([]map[string]any)
	if !ok || len(ch) != 3 {
		t.Fatalf("choices=%v", res["choices"])
	}
}

func TestRunOverallDateEchoUnchanged(t *testing.T) {
	srv := overallJEVSrv(t)
	defer srv.Close()
	const orch = "2026-11-22"
	res, err := diagnoseskill.RunOverall(context.Background(), diagnoseskill.MacroOverallRunInput{
		Date: orch,
		Pools: []diagnoseskill.MacroOverallPoolRow{
			{PoolWorkflow: "crypto", JEVDirection: "short", JEVReason: cryptoReason},
		},
		JEV: strategydev.NewJevClient(strategydev.JevConfig{BaseURL: srv.URL, APIKey: "k"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res["date"] != orch {
		t.Fatalf("date=%v", res["date"])
	}
}

func TestRunOverallChoiceFailureNoFakeSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	_, err := diagnoseskill.RunOverall(context.Background(), diagnoseskill.MacroOverallRunInput{
		Date: "2026-11-22",
		Pools: []diagnoseskill.MacroOverallPoolRow{
			{PoolWorkflow: "crypto", JEVDirection: "short", JEVReason: cryptoReason},
		},
		JEV: strategydev.NewJevClient(strategydev.JevConfig{BaseURL: srv.URL, APIKey: "k"}),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}