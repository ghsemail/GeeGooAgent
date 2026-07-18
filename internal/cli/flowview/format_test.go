package flowview_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/cli/flowview"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
)

func TestFormatRunCompleted(t *testing.T) {
	line := flowview.Format(infra.EventRecord{
		Event: "RunCompleted",
		Payload: map[string]any{
			"skill": "pre_market", "session_id": "sess-1", "status": "completed", "verdict": "pass",
		},
	})
	if !strings.Contains(line, "Skill 完成") || !strings.Contains(line, "pre_market") {
		t.Fatalf("got %q", line)
	}
}

func TestFormatSynthesisStarted(t *testing.T) {
	line := flowview.Format(infra.EventRecord{
		Event: "SynthesisStarted",
		Payload: map[string]any{
			"code": "00700.HK", "stock_name": "腾讯控股", "evidence_count": 3,
		},
	})
	if !strings.Contains(line, "报告合成开始") || !strings.Contains(line, "00700.HK") {
		t.Fatalf("got %q", line)
	}
}
