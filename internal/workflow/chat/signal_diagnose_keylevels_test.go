package chat

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestKeyLevelSnapshotFromProbeDataFlat(t *testing.T) {
	kl := keyLevelSnapshotFromProbeData(map[string]any{
		"support_low": 50.0, "support_high": 52.0, "resist_high": 110.0,
		"summary": "probe snapshot",
	})
	if kl.SupportLow != 50 || kl.ResistHigh != 110 || kl.Summary != "probe snapshot" {
		t.Fatalf("kl=%+v", kl)
	}
	if !keyLevelSnapshotUsableForEpisodeStop(kl) {
		t.Fatal("expected usable")
	}
}

func TestSignalDiagnoseProbeSkipsGetKeyLevelsWhenEmbedded(t *testing.T) {
	var keyLevelCalls int
	runner := signalDiagnoseTestRunner(t)
	runner.RunTool = func(ctx context.Context, req tools.CallRequest, tc tools.Context) tools.Result {
		if req.Name == "get_key_levels" {
			keyLevelCalls++
		}
		if req.Name == "probe_bot_signal_series" {
			if v, _ := req.Arguments["include_key_levels"].(bool); !v {
				t.Fatal("expected include_key_levels=true on probe")
			}
			if req.Arguments["key_levels_mode"] != "series" {
				t.Fatalf("expected key_levels_mode=series got %v", req.Arguments["key_levels_mode"])
			}
			res := signalDiagnoseTestRunner(t).RunTool(ctx, req, tc)
			res.Data["key_levels"] = map[string]any{
				"support_low": 50.0, "resist_high": 110.0, "summary": "from probe",
			}
			return res
		}
		if req.Name == "get_signal_combinations" {
			return tools.Result{Status: tools.StatusError, Summary: "not found"}
		}
		if req.Name == "get_index_signals" {
			return tools.Result{Status: tools.StatusOK, Data: map[string]any{"items": []any{
				map[string]any{
					"name": "SAR", "index": "SAR", "frequency": "60m",
					"buy_signal": []any{map[string]any{
						"index": "SAR", "type": "signal", "param": map[string]any{"acceleration": 0.02, "maximum": 0.2},
					}},
					"sell_signal": []any{map[string]any{
						"index": "SAR", "type": "signal", "param": map[string]any{"acceleration": 0.02, "maximum": 0.2},
					}},
				},
			}}}
		}
		return signalDiagnoseTestRunner(t).RunTool(ctx, req, tc)
	}
	session := runtime.NewSession()
	session.PendingSignalDiagnoseOpts = &runtime.SignalDiagnoseOpts{
		UseKeyLevelEpisodeStop: true,
		KeyBreakMode:           KeyBreakModeSupportLow,
	}
	result, handled := runner.RunTurn(context.Background(), session, "诊断 SAR 策略 · 腾讯", tools.Context{}, 8)
	if !handled || result.Failed {
		t.Fatalf("handled=%v failed=%v err=%s", handled, result.Failed, result.Error)
	}
	if keyLevelCalls != 0 {
		t.Fatalf("get_key_levels calls=%d want 0", keyLevelCalls)
	}
}

func TestSignalDiagnoseProbeFallsBackGetKeyLevels(t *testing.T) {
	var keyLevelCalls int
	runner := signalDiagnoseTestRunner(t)
	runner.RunTool = func(ctx context.Context, req tools.CallRequest, tc tools.Context) tools.Result {
		if req.Name == "get_key_levels" {
			keyLevelCalls++
		}
		if req.Name == "get_signal_combinations" {
			return tools.Result{Status: tools.StatusError, Summary: "not found"}
		}
		if req.Name == "get_index_signals" {
			return tools.Result{Status: tools.StatusOK, Data: map[string]any{"items": []any{
				map[string]any{
					"name": "SAR", "index": "SAR", "frequency": "60m",
					"buy_signal": []any{map[string]any{
						"index": "SAR", "type": "signal", "param": map[string]any{"acceleration": 0.02, "maximum": 0.2},
					}},
					"sell_signal": []any{map[string]any{
						"index": "SAR", "type": "signal", "param": map[string]any{"acceleration": 0.02, "maximum": 0.2},
					}},
				},
			}}}
		}
		return signalDiagnoseTestRunner(t).RunTool(ctx, req, tc)
	}
	session := runtime.NewSession()
	session.PendingSignalDiagnoseOpts = &runtime.SignalDiagnoseOpts{
		UseKeyLevelEpisodeStop: true,
		KeyBreakMode:           KeyBreakModeSupportLow,
	}
	_, handled := runner.RunTurn(context.Background(), session, "诊断 SAR 策略 · 腾讯", tools.Context{}, 8)
	if !handled {
		t.Fatal("expected handled")
	}
	if keyLevelCalls != 1 {
		t.Fatalf("get_key_levels calls=%d want 1 fallback", keyLevelCalls)
	}
}
