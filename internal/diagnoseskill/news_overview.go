package diagnoseskill

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

// diagnose_macro_news_overview concludes one day's macro news across pools. It takes
// the pool verdicts and a few item summaries, not the articles. It writes nothing.

const (
	SkillMacroNewsOverview = "diagnose_macro_news_overview"

	maxOverviewItemsPerPool = 12
)

// MacroNewsOverviewItem is a compact item summary.
type MacroNewsOverviewItem struct {
	Title        string `json:"title"`
	JEVDirection string `json:"jev_direction"`
	JEVReason    string `json:"jev_reason"`
}

// MacroNewsOverviewPool is one finished pool verdict.
type MacroNewsOverviewPool struct {
	PoolID         string                  `json:"pool_id"`
	PoolLabel      string                  `json:"pool_label"`
	JEVDirection   string                  `json:"jev_direction"`
	JEVReason      string                  `json:"jev_reason"`
	Bullish        int                     `json:"bullish"`
	Bearish        int                     `json:"bearish"`
	Neutral        int                     `json:"neutral"`
	HighValueCount int                     `json:"high_value_count"`
	Items          []MacroNewsOverviewItem `json:"items"`
}

// MacroNewsOverviewRequest is the macro_news_overview object of POST /v1/skills/run.
type MacroNewsOverviewRequest struct {
	Date   string                  `json:"date"`
	NodeID string                  `json:"node_id"`
	RunID  string                  `json:"run_id"`
	Pools  []MacroNewsOverviewPool `json:"pools"`
	// MissingPools have no verdict for the date; the conclusion says so.
	MissingPools []string `json:"missing_pools"`
}

// MacroNewsOverviewRunInput is the validated request plus its clients.
type MacroNewsOverviewRunInput struct {
	Request MacroNewsOverviewRequest
	LLM     MacroNewsLLM
	JEV     *strategydev.JevClient
}

// ValidateMacroNewsOverviewRequest fails closed on a malformed request.
func ValidateMacroNewsOverviewRequest(in MacroNewsOverviewRequest) error {
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(in.Date)); err != nil {
		return fmt.Errorf("invalid macro_news_overview.date")
	}
	if len(in.Pools) == 0 {
		return fmt.Errorf("macro_news_overview.pools is required")
	}
	seen := map[string]bool{}
	unique := func(id string) error {
		if _, ok := macroPoolLabels[id]; !ok {
			return fmt.Errorf("invalid macro_news_overview pool_id %q", id)
		}
		if seen[id] {
			return fmt.Errorf("duplicate macro_news_overview pool_id %s", id)
		}
		seen[id] = true
		return nil
	}
	for _, pool := range in.Pools {
		if err := unique(pool.PoolID); err != nil {
			return err
		}
		// A pool without a direction belongs in missing_pools; it is not neutral.
		if _, ok := normalizeDirection(pool.JEVDirection); !ok {
			return fmt.Errorf("invalid jev_direction for pool %s", pool.PoolID)
		}
		if len(pool.Items) > maxOverviewItemsPerPool {
			return fmt.Errorf("pool %s allows at most %d items", pool.PoolID, maxOverviewItemsPerPool)
		}
	}
	for _, id := range in.MissingPools {
		if err := unique(id); err != nil {
			return err
		}
	}
	return nil
}

// RunMacroNewsOverview returns the overall direction (Decision Choice) and its
// conclusion. A failed Choice is an error, never a neutral result.
func RunMacroNewsOverview(ctx context.Context, in MacroNewsOverviewRunInput) (Result, error) {
	req := in.Request
	req.Date = strings.TrimSpace(req.Date)
	if err := ValidateMacroNewsOverviewRequest(req); err != nil {
		return nil, err
	}
	var pools []strategydev.MacroNewsOverviewPool
	for _, pool := range req.Pools {
		direction, _ := normalizeDirection(pool.JEVDirection)
		entry := strategydev.MacroNewsOverviewPool{
			PoolID: pool.PoolID, Label: macroPoolLabels[pool.PoolID], Direction: direction, Reason: pool.JEVReason,
			Bullish: pool.Bullish, Bearish: pool.Bearish, Neutral: pool.Neutral, HighValueCount: pool.HighValueCount,
		}
		for _, item := range pool.Items {
			itemDirection, _ := normalizeDirection(item.JEVDirection)
			entry.Items = append(entry.Items, strategydev.MacroNewsPoolItem{
				Title: item.Title, JEVDirection: itemDirection, JEVReason: item.JEVReason,
			})
		}
		pools = append(pools, entry)
	}
	missing := append([]string{}, req.MissingPools...)
	labels := make([]string, len(missing))
	for i, id := range missing {
		labels[i] = macroPoolLabels[id]
	}

	runCtx, cancel := context.WithTimeout(ctx, macroNewsOverviewBudget)
	defer cancel()
	var summarizer strategydev.MacroNewsSummarizer
	if in.LLM != nil {
		summarizer = in.LLM
	}
	verdict := strategydev.ResolveMacroNewsOverviewJEV(runCtx, in.JEV, req.Date, pools, labels, summarizer)
	if !verdict.OK {
		return nil, fmt.Errorf("macro news overview: %s", clipRunes(verdict.Error, 200))
	}
	direction := DirectionToLedger(string(verdict.Direction))
	if _, ok := normalizeDirection(direction); !ok {
		return nil, fmt.Errorf("macro news overview: verdict invalid")
	}
	return Result{
		"date": req.Date, "node_id": req.NodeID, "run_id": req.RunID, "detail_ref": SkillMacroNewsOverview,
		"model": verdict.Model, "jev_direction": direction, "jev_reason": verdict.RecommendedReason,
		"reason_status": verdict.ReasonStatus, "confidence": verdict.Confidence,
		"choices": poolChoicesForResult(verdict.Choices), "conflict": verdict.Conflict,
		"missing_pools": missing,
	}, nil
}
