package premarket

import (
	"context"
	"fmt"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/stockfmt"
)

// SignalKeyLevelFetcher calls GeeGooSignal Key Level Engine via /getSupportingPrice.
type SignalKeyLevelFetcher struct {
	Signal *mcp.Client
}

// FetchKeyLevels implements keylevelctx.Fetcher.
func (f SignalKeyLevelFetcher) FetchKeyLevels(ctx context.Context, code string) (stockfmt.KeyLevels, error) {
	if f.Signal == nil {
		return stockfmt.KeyLevels{}, fmt.Errorf("signal api client not configured")
	}
	resp, err := f.Signal.GetSupportingPrice(ctx, code, map[string]any{"include_legacy": true})
	if err != nil {
		return stockfmt.KeyLevels{}, err
	}
	return keyLevelsFromSupportingPrice(resp), nil
}

func keyLevelsFromSupportingPrice(resp *mcp.SupportingPriceResponse) stockfmt.KeyLevels {
	if resp == nil {
		return stockfmt.KeyLevels{}
	}
	data := resp.Data
	levels := stockfmt.KeyLevels{Provenance: stockfmt.KeyLevelProvenance{EngineUsed: true}}

	var pickSupport, pickResistance float64
	var supportZone, resistZone string
	var supportSrc, resistSrc []string

	if data.Judgment.Support != nil {
		j := data.Judgment.Support
		pickSupport = j.Center
		supportZone = fmt.Sprintf("%.2f~%.2f", j.Low, j.High)
		supportSrc = append([]string(nil), j.Sources...)
	} else if data.Candidates != nil && len(data.Candidates.Support) > 0 {
		z := data.Candidates.Support[0]
		pickSupport = z.Center
		supportZone = fmt.Sprintf("%.2f~%.2f", z.Low, z.High)
	} else if data.Legacy != nil && data.Legacy.QFLSupport > 0 {
		pickSupport = data.Legacy.QFLSupport
	}
	if data.Judgment.Resistance != nil {
		j := data.Judgment.Resistance
		pickResistance = j.Center
		resistZone = fmt.Sprintf("%.2f~%.2f", j.Low, j.High)
		resistSrc = append([]string(nil), j.Sources...)
	} else if data.Candidates != nil && len(data.Candidates.Resistance) > 0 {
		z := data.Candidates.Resistance[0]
		pickResistance = z.Center
		resistZone = fmt.Sprintf("%.2f~%.2f", z.Low, z.High)
	} else if data.Legacy != nil && data.Legacy.QFLResistance > 0 {
		pickResistance = data.Legacy.QFLResistance
	}
	if pickSupport > 0 {
		s := stockfmt.RoundPrice(pickSupport)
		levels.Support = &s
	}
	if pickResistance > 0 {
		r := stockfmt.RoundPrice(pickResistance)
		levels.Resistance = &r
	}
	levels.Valid = levels.Support != nil && levels.Resistance != nil
	levels.Provenance.SupportZone = supportZone
	levels.Provenance.ResistZone = resistZone
	levels.Provenance.SupportSrc = supportSrc
	levels.Provenance.ResistSrc = resistSrc
	return levels
}
