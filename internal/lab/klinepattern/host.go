package klinepattern

import (
	"context"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// ToolRunner invokes MCP tools (typically probe_bot_signal_series).
type ToolRunner func(ctx context.Context, name string, args map[string]any) tools.Result

// Host wires production strategydev helpers into archived lab code without import cycles.
type Host struct {
	ProbeArgsExtended           func(code, freq string, months int, asOf *time.Time) map[string]any
	MonthsBackForPerSignalFetch func(chartMonths int, chartBars []any, freq string, windowEnd time.Time) int
}

var host Host

// SetHost registers callbacks from strategydev (call once at process init).
func SetHost(h Host) {
	host = h
}
