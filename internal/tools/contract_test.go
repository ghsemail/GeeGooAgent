package tools_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestClassifyEmptyListAsSkip(t *testing.T) {
	t.Parallel()
	status, note, gap := tools.ClassifyHTTPPayload("list_smart_trades",
		map[string]any{"items": []any{}, "count": 0}, nil)
	if status != tools.StatusSkip {
		t.Fatalf("expected skip, got %s", status)
	}
	if !gap {
		t.Fatal("expected data gap flag")
	}
	if note == "" {
		t.Fatal("expected non-empty note")
	}
}

func TestClassifyNonEmptyAsOK(t *testing.T) {
	t.Parallel()
	status, _, _ := tools.ClassifyHTTPPayload("list_smart_trades",
		map[string]any{"items": []any{map[string]any{"bot_id": "b1"}}, "count": 1}, nil)
	if status != tools.StatusOK {
		t.Fatalf("expected ok, got %s", status)
	}
}

func TestClassifyEmptyAnalysisResultAsSkip(t *testing.T) {
	t.Parallel()
	status, _, _ := tools.ClassifyHTTPPayload("get_mcp_analysis",
		map[string]any{"code": "00700.HK", "period": "weekly", "analysis_result": ""}, nil)
	if status != tools.StatusSkip {
		t.Fatalf("expected skip for empty analysis, got %s", status)
	}
}

func TestClassifyNonEmptyToolAlwaysOK(t *testing.T) {
	t.Parallel()
	// get_current_price is not in EmptyResultTools; even nil payload stays OK.
	status, _, _ := tools.ClassifyHTTPPayload("get_current_price", nil, nil)
	if status != tools.StatusOK {
		t.Fatalf("expected ok for non-listed tool, got %s", status)
	}
}

func TestClassifyEmptyGridStrategyAsSkip(t *testing.T) {
	t.Parallel()
	status, note, gap := tools.ClassifyHTTPPayload("generate_grid_strategy",
		map[string]any{"suitable": false, "reason": "not suitable"}, nil)
	if status != tools.StatusSkip {
		t.Fatalf("expected skip, got %s", status)
	}
	if !gap || note == "" {
		t.Fatalf("expected data gap note, got gap=%v note=%q", gap, note)
	}
}

func TestClassifyGridStrategyWithParamAsOK(t *testing.T) {
	t.Parallel()
	status, _, _ := tools.ClassifyHTTPPayload("generate_grid_strategy", map[string]any{
		"code": "00700.HK",
		"param": map[string]any{
			"upper_limit_price": 640.0,
			"lower_limit_price": 500.0,
			"grid_num":          7.0,
		},
		"suitable": true,
	}, nil)
	if status != tools.StatusOK {
		t.Fatalf("expected ok, got %s", status)
	}
}

func TestClassifyEmptyDCAStrategyAsSkip(t *testing.T) {
	t.Parallel()
	status, _, _ := tools.ClassifyHTTPPayload("generate_dca_strategy",
		map[string]any{"signal": map[string]any{"buy_signal": []any{}}}, nil)
	if status != tools.StatusSkip {
		t.Fatalf("expected skip, got %s", status)
	}
}

func TestClassifyDCAStrategyWithBuySignalAsOK(t *testing.T) {
	t.Parallel()
	status, _, _ := tools.ClassifyHTTPPayload("generate_dca_strategy", map[string]any{
		"code": "00700.HK",
		"signal": map[string]any{
			"buy_signal": []any{map[string]any{"index": "SAR"}},
		},
	}, nil)
	if status != tools.StatusOK {
		t.Fatalf("expected ok, got %s", status)
	}
}
