package lab

import (
	"context"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/lab/priceanalysis"
	"github.com/ghsemail/GeeGooAgent/internal/lab/signalclient"
)

// SignalBarsFetcher implements priceanalysis.BarsFetcher via GeeGooSignal lab-api.
type SignalBarsFetcher struct {
	Client signalclient.Client
}

func (f SignalBarsFetcher) FetchBars(ctx context.Context, code, freq string, months int, asOf *time.Time) ([]any, bool) {
	if f.Client == nil {
		return nil, false
	}
	req := signalclient.KlineJobRequest{
		Code: code, Frequency: freq, MonthsBack: months, AsOf: asOf,
	}
	if hc, ok := f.Client.(*signalclient.HTTPClient); ok {
		bars, err := hc.FetchBars(ctx, req)
		if err != nil {
			return nil, false
		}
		return bars, true
	}
	const attempts = 3
	var lastOK bool
	for attempt := 0; attempt < attempts; attempt++ {
		if ctx.Err() != nil {
			return nil, false
		}
		bars, ok := f.fetchBarsViaJobAPI(ctx, req)
		if ok {
			return bars, true
		}
		lastOK = ok
		if attempt+1 >= attempts {
			break
		}
		time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
	}
	_ = lastOK
	return nil, false
}

func (f SignalBarsFetcher) fetchBarsViaJobAPI(ctx context.Context, req signalclient.KlineJobRequest) ([]any, bool) {
	jobID, err := f.Client.CreateKlineJob(ctx, req)
	if err != nil {
		return nil, false
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		job, err := f.Client.GetJob(ctx, jobID)
		if err != nil {
			return nil, false
		}
		switch job.Status {
		case signalclient.JobCompleted:
			if job.Result == nil || job.Result.Bars == nil {
				return nil, false
			}
			return job.Result.Bars, true
		case signalclient.JobFailed:
			return nil, false
		default:
			if time.Now().After(deadline) {
				return nil, false
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

var _ priceanalysis.BarsFetcher = SignalBarsFetcher{}
