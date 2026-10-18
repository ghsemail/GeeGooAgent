package lab

import (
	"context"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/lab/priceanalysis"
	"github.com/ghsemail/GeeGooAgent/internal/lab/signalclient"
)

const ExperimentPriceAnalysis = "price-analysis"

// Service runs Agent lab experiments (isolated from diagnose / strategydev).
type Service struct {
	Fetcher priceanalysis.BarsFetcher
}

// NewService wires the default Signal lab-api fetcher.
func NewService(client signalclient.Client) *Service {
	return &Service{Fetcher: SignalBarsFetcher{Client: client}}
}

// PriceAnalysisRequest is the HTTP/API input for the price-analysis experiment.
type PriceAnalysisRequest struct {
	Code           string     `json:"code"`
	SignalFrequency string    `json:"signal_frequency"`
	Months         int        `json:"months,omitempty"`
	AnchorTime     *time.Time `json:"anchor_time,omitempty"`
	SpotPrice      float64    `json:"spot_price,omitempty"`
}

// PriceAnalysisResponse wraps analysis output for lab routes.
type PriceAnalysisResponse struct {
	Experiment string                    `json:"experiment"`
	Analysis   priceanalysis.PriceAnalysis `json:"analysis"`
}

// RunPriceAnalysis executes the migrated multi-TF price analysis experiment.
func (s *Service) RunPriceAnalysis(ctx context.Context, req PriceAnalysisRequest) (PriceAnalysisResponse, error) {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return PriceAnalysisResponse{}, errBadRequest("code required")
	}
	freq := strings.TrimSpace(req.SignalFrequency)
	if freq == "" {
		freq = "60m"
	}
	fetch := s.Fetcher
	if fetch == nil {
		fetch = SignalBarsFetcher{}
	}
	pa := priceanalysis.Run(ctx, fetch, priceanalysis.RunRequest{
		Code:       code,
		Months:     req.Months,
		SignalFreq: freq,
		AsOf:       req.AnchorTime,
		SpotPrice:  req.SpotPrice,
	})
	pa = priceanalysis.RoundPriceAnalysisForOutput(pa)
	return PriceAnalysisResponse{Experiment: ExperimentPriceAnalysis, Analysis: pa}, nil
}

type badRequestError string

func (e badRequestError) Error() string { return string(e) }

func errBadRequest(msg string) error { return badRequestError(msg) }

// IsBadRequest reports validation errors from lab service methods.
func IsBadRequest(err error) bool {
	_, ok := err.(badRequestError)
	return ok
}
