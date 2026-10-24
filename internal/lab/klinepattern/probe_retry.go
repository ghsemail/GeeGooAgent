package klinepattern

import (
	"context"
	"time"
)

const klinePatternProbeAttempts = 3

func fetchKlinePatternProbeSeriesWithRetry(
	ctx context.Context,
	run ToolRunner,
	code, freq string,
	months int,
	asOf *time.Time,
) (bars, patternHits []any, ok, unsupported bool) {
	var lastOK bool
	for attempt := 0; attempt < klinePatternProbeAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, nil, false, false
		}
		b, h, o, u := fetchKlinePatternProbeSeries(ctx, run, code, freq, months, asOf)
		if u {
			return b, h, false, true
		}
		if o {
			return b, h, true, false
		}
		lastOK = o
		if attempt+1 >= klinePatternProbeAttempts {
			break
		}
		timer := time.NewTimer(time.Duration(attempt+1) * 200 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, nil, false, false
		case <-timer.C:
		}
	}
	return nil, nil, lastOK, false
}
