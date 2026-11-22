package diagnoseskill

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

// LaneJEV is the slim per-lane JEV block in diagnose skill responses.
type LaneJEV struct {
	OK                bool     `json:"ok"`
	Direction         string   `json:"direction,omitempty"`
	Choices           []choice `json:"choices,omitempty"`
	RecommendedReason string   `json:"recommended_reason,omitempty"`
	Error             string   `json:"error,omitempty"`
}

type choice struct {
	Label       string  `json:"label"`
	Probability float64 `json:"probability"`
}

// UnifiedJEV is total JEV in diagnose skill responses.
type UnifiedJEV struct {
	LaneJEV
	Confidence float64 `json:"confidence,omitempty"`
	Conflict   bool    `json:"conflict,omitempty"`
}

func emitLaneJEV(pa strategydev.PriceAnalysisJEV) LaneJEV {
	return laneFromInternal(pa.OK, pa.Error, pa.RecommendedChoice, pa.Choices, pa.RecommendedReason)
}

func emitKlineJEV(k strategydev.KlineJEV) LaneJEV {
	return laneFromInternal(k.OK, k.Error, k.RecommendedChoice, k.Choices, k.RecommendedReason)
}

func emitTAJEV(t strategydev.TechnicalAnalysisJEV) LaneJEV {
	return laneFromInternal(t.OK, t.Error, t.RecommendedChoice, t.Choices, t.RecommendedReason)
}

func emitIndexJEV(i strategydev.IndexTrendJEV) LaneJEV {
	return laneFromInternal(i.OK, i.Error, i.RecommendedChoice, i.Choices, i.RecommendedReason)
}

func emitUnifiedJEV(u *strategydev.LabJEVUnified) UnifiedJEV {
	if u == nil {
		return UnifiedJEV{LaneJEV: LaneJEV{OK: false, Error: "jev_unified absent"}}
	}
	base := laneFromInternal(u.OK, u.Error, u.RecommendedChoice, u.Choices, u.RecommendedReason)
	if !u.OK {
		return UnifiedJEV{LaneJEV: base}
	}
	dir := string(u.Direction)
	if dir == "" {
		dir = labelToDirection(u.RecommendedChoice)
	}
	base.Direction = dir
	return UnifiedJEV{
		LaneJEV:    base,
		Confidence: u.Confidence,
		Conflict:   u.Conflict,
	}
}

func laneFromInternal(ok bool, errMsg, recommended string, choices []strategydev.PriceAnalysisJEVChoice, reason string) LaneJEV {
	out := LaneJEV{OK: ok, Error: strings.TrimSpace(errMsg), RecommendedReason: strings.TrimSpace(reason)}
	if !ok {
		out.Direction = ""
		out.Choices = nil
		return out
	}
	out.Direction = labelToDirection(recommended)
	for _, c := range choices {
		out.Choices = append(out.Choices, choice{Label: c.Label, Probability: c.Probability})
	}
	return out
}

func labelToDirection(label string) string {
	switch strings.TrimSpace(label) {
	case "看多":
		return "bull"
	case "看空":
		return "bear"
	default:
		return "unclear"
	}
}