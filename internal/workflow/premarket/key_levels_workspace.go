package premarket

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/stockfmt"
)

func keyLevelsFromWorkspace(ws memory.StockWorkspace) stockfmt.KeyLevels {
	if !ws.KeyLevelsEngineOK {
		return stockfmt.KeyLevels{}
	}
	out := stockfmt.KeyLevels{
		Valid: ws.KeyLevelSupportCenter > 0 && ws.KeyLevelResistanceCenter > 0,
		Provenance: stockfmt.KeyLevelProvenance{
			EngineUsed:  true,
			SupportZone: ws.KeyLevelSupportZone,
			ResistZone:  ws.KeyLevelResistZone,
		},
	}
	if ws.KeyLevelSupportCenter > 0 {
		s := stockfmt.RoundPrice(ws.KeyLevelSupportCenter)
		out.Support = &s
	}
	if ws.KeyLevelResistanceCenter > 0 {
		r := stockfmt.RoundPrice(ws.KeyLevelResistanceCenter)
		out.Resistance = &r
	}
	if ws.KeyLevelSupportSources != "" {
		out.Provenance.SupportSrc = strings.Split(ws.KeyLevelSupportSources, ",")
	}
	if ws.KeyLevelResistSources != "" {
		out.Provenance.ResistSrc = strings.Split(ws.KeyLevelResistSources, ",")
	}
	return out
}
