package strategydev

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

type stubKlineProbeRunner struct {
	calls []string
	fail4h bool
}

func (s *stubKlineProbeRunner) Run(ctx context.Context, name string, args map[string]any) tools.Result {
	if name != "probe_bot_signal_series" {
		return tools.Result{Status: tools.StatusError}
	}
	freq, _ := args["frequency"].(string)
	s.calls = append(s.calls, freq)
	if freq == "240m" && s.fail4h {
		return tools.Result{Status: tools.StatusError, Summary: "unsupported frequency"}
	}
	return tools.Result{
		Status: tools.StatusOK,
		Data: map[string]any{
			"bars": []any{map[string]any{"time": "2024-01-01", "close": 1}},
		},
	}
}

func TestBuildKlinePatternMTFBundle_FourHourProbeAndFailure(t *testing.T) {
	SetKlinePatternDisplayEnabledForTest(t, true)
	runOK := &stubKlineProbeRunner{}
	bundle := BuildKlinePatternMTFBundle(context.Background(), runOK.Run, "00700.HK", "60m", 3, nil)
	if bundle == nil {
		t.Fatal("nil bundle")
	}
	var saw240m bool
	for _, f := range runOK.calls {
		if f == "240m" {
			saw240m = true
		}
	}
	if !saw240m {
		t.Fatalf("expected 240m probe for 4h ladder, calls=%v", runOK.calls)
	}
	ser, ok := bundle.ByFreq["4h"]
	if !ok {
		t.Fatalf("ByFreq keys=%v", bundle.ByFreq)
	}
	if !ser.FetchOK {
		t.Fatal("expected 4h fetch ok")
	}

	runFail := &stubKlineProbeRunner{fail4h: true}
	bundle2 := BuildKlinePatternMTFBundle(context.Background(), runFail.Run, "00700.HK", "1h", 3, nil)
	ser2 := bundle2.ByFreq["4h"]
	if ser2.FetchOK {
		t.Fatal("expected 4h fetch failure")
	}
}
