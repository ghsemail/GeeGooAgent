package mcp

import (
	"context"
	"encoding/json"
)

// SupportingPriceSideJudgment primary support/resistance band.
type SupportingPriceSideJudgment struct {
	Center     float64  `json:"center"`
	Low        float64  `json:"low"`
	High       float64  `json:"high"`
	Score      int      `json:"score"`
	State      string   `json:"state"`
	Confidence string   `json:"confidence"`
	Sources    []string `json:"sources"`
}

// SupportingPriceJudgment code conclusion.
type SupportingPriceJudgment struct {
	Support    *SupportingPriceSideJudgment `json:"support"`
	Resistance *SupportingPriceSideJudgment `json:"resistance"`
	Regime     string                       `json:"regime"`
	Summary    string                       `json:"summary"`
}

// SupportingPriceCandidateZone secondary zone.
type SupportingPriceCandidateZone struct {
	Center float64 `json:"center"`
	Low    float64 `json:"low"`
	High   float64 `json:"high"`
	Score  int     `json:"score"`
	State  string  `json:"state"`
}

// SupportingPriceCandidates optional secondary bands.
type SupportingPriceCandidates struct {
	Support    []SupportingPriceCandidateZone `json:"support"`
	Resistance []SupportingPriceCandidateZone `json:"resistance"`
}

// LevelEvidenceJSON clustered candidate price.
type LevelEvidenceJSON struct {
	Type      string  `json:"type"`
	Price     float64 `json:"price"`
	Timeframe string  `json:"timeframe"`
}

// SupportingPriceRefEvidence evidence for primary bands.
type SupportingPriceRefEvidence struct {
	Support    []LevelEvidenceJSON `json:"support"`
	Resistance []LevelEvidenceJSON `json:"resistance"`
}

// SupportingPriceRefs minimal refs.
type SupportingPriceRefs struct {
	Bars        SupportingPriceBarMeta     `json:"bars"`
	ATR14       float64                    `json:"atr14"`
	Pivot       *PivotLevels               `json:"pivot"`
	Evidence    SupportingPriceRefEvidence `json:"evidence"`
	CapitalHint string                     `json:"capital_hint"`
}

// SupportingPriceBarMeta bar counts.
type SupportingPriceBarMeta struct {
	DailyBars  int `json:"daily_bars"`
	WeeklyBars int `json:"weekly_bars"`
	Hour60Bars int `json:"hour60_bars"`
}

// PivotLevels floor pivot reference.
type PivotLevels struct {
	P  float64 `json:"P"`
	S1 float64 `json:"S1"`
	R1 float64 `json:"R1"`
	S2 float64 `json:"S2"`
	R2 float64 `json:"R2"`
}

// SupportingPriceLegacy grid-app fields.
type SupportingPriceLegacy struct {
	QFLSupport      float64 `json:"QFLSupport"`
	QFLResistance   float64 `json:"QFLResistance"`
	BBANDSupport    float64 `json:"BBANDSupport"`
	BBANDResistance float64 `json:"BBANDResistance"`
	SARSupport      float64 `json:"SARSupport"`
	SARResistance   float64 `json:"SARResistance"`
	High            float64 `json:"high"`
	Low             float64 `json:"low"`
	TradeDate       string  `json:"trade_date"`
}

// SupportingPriceData response payload.
type SupportingPriceData struct {
	CurrentPrice float64                    `json:"current_price"`
	Judgment     SupportingPriceJudgment    `json:"judgment"`
	Candidates   *SupportingPriceCandidates `json:"candidates"`
	Refs         SupportingPriceRefs        `json:"refs"`
	Legacy       *SupportingPriceLegacy     `json:"legacy"`
}

// SupportingPriceResponse envelope.
type SupportingPriceResponse struct {
	Code string              `json:"code"`
	Data SupportingPriceData `json:"data"`
}

// GetSupportingPrice calls POST /getSupportingPrice.
func (c *Client) GetSupportingPrice(ctx context.Context, code string, opts map[string]any) (*SupportingPriceResponse, error) {
	body := map[string]any{"code": code, "include_60m": true}
	for k, v := range opts {
		body[k] = v
	}
	raw, err := c.PostDirect(ctx, "/getSupportingPrice", body)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var out SupportingPriceResponse
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
