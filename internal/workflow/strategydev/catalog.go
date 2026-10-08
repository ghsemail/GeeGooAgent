package strategydev

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// ToolRunner executes catalog / probe tools.
type ToolRunner func(ctx context.Context, name string, args map[string]any) tools.Result

// Catalog loads index + combination + advanced strategies from signal catalog tools.
type Catalog struct {
	Indices      []Candidate
	Combinations []Candidate
	Advanced     []Candidate
}

func LoadCatalog(ctx context.Context, run ToolRunner) (Catalog, error) {
	var out Catalog
	idxRes := run(ctx, "get_index_signals", map[string]any{})
	if idxRes.Status == tools.StatusOK {
		for _, row := range slots.CatalogItems(idxRes.Data) {
			c, err := candidateFromIndexRow(row)
			if err == nil {
				out.Indices = append(out.Indices, c)
			}
		}
	}
	comboRes := run(ctx, "get_signal_combinations", map[string]any{})
	if comboRes.Status == tools.StatusOK {
		for _, row := range slots.CatalogItems(comboRes.Data) {
			c, err := candidateFromCombinationRow(row)
			if err == nil {
				out.Combinations = append(out.Combinations, c)
			}
		}
	}
	advRes := run(ctx, "get_agent_signal_combination", map[string]any{})
	if advRes.Status == tools.StatusOK {
		for _, row := range slots.CatalogItems(advRes.Data) {
			c, err := candidateFromAdvancedRow(row)
			if err == nil {
				out.Advanced = append(out.Advanced, c)
			}
		}
	}
	customRes := run(ctx, "get_custom_signal_for_skill", map[string]any{})
	if customRes.Status == tools.StatusOK {
		for _, row := range slots.CatalogItems(customRes.Data) {
			c, err := candidateFromAdvancedRow(row)
			if err == nil {
				out.Advanced = append(out.Advanced, c)
			}
		}
	}
	if len(out.Indices) == 0 && len(out.Combinations) == 0 && len(out.Advanced) == 0 {
		return out, fmt.Errorf("策略目录为空")
	}
	return out, nil
}

func candidateFromIndexRow(row map[string]any) (Candidate, error) {
	id := strings.TrimSpace(fmt.Sprint(row["signal_id"]))
	if id == "" || id == "<nil>" {
		return Candidate{}, fmt.Errorf("missing signal_id")
	}
	buy, sell, _, label, err := indexRulesFromRow(row)
	if err != nil {
		return Candidate{}, err
	}
	return Candidate{
		SignalID: id,
		Name:     label,
		Kind:     "index",
		Buy:      buy,
		Sell:     sell,
	}, nil
}

func indexRulesFromRow(row map[string]any) (buy, sell []any, frequency, label string, err error) {
	if rawBuy, ok := row["buy_signal"].([]any); ok && len(rawBuy) > 0 {
		buy = rawBuy
	} else {
		idx := strings.TrimSpace(fmt.Sprint(row["index"]))
		if idx == "" {
			return nil, nil, "", "", fmt.Errorf("index row missing rules")
		}
		buy = []any{map[string]any{"index": idx, "type": "signal", "param": row["param"]}}
	}
	sell = buy
	if rawSell, ok := row["sell_signal"].([]any); ok && len(rawSell) > 0 {
		sell = rawSell
	}
	label = strings.TrimSpace(fmt.Sprint(row["name"]))
	frequency = strings.TrimSpace(fmt.Sprint(row["frequency"]))
	return buy, sell, frequency, label, nil
}

func candidateFromCombinationRow(row map[string]any) (Candidate, error) {
	id := strings.TrimSpace(fmt.Sprint(row["signal_id"]))
	buy, _ := row["buy_signal"].([]any)
	if id == "" || len(buy) == 0 {
		return Candidate{}, fmt.Errorf("invalid combination")
	}
	sell, _ := row["sell_signal"].([]any)
	return Candidate{
		SignalID: id,
		Name:     strings.TrimSpace(fmt.Sprint(row["name"])),
		Kind:     "combination",
		Buy:      buy,
		Sell:     sell,
	}, nil
}

func candidateFromAdvancedRow(row map[string]any) (Candidate, error) {
	id := strings.TrimSpace(fmt.Sprint(row["signal_id"]))
	if id == "" {
		return Candidate{}, fmt.Errorf("missing id")
	}
	buy, _ := row["buy_signal"].([]any)
	sell, _ := row["sell_signal"].([]any)
	name := strings.TrimSpace(fmt.Sprint(row["name"]))
	if name == "" {
		name = strings.TrimSpace(fmt.Sprint(row["brief"]))
	}
	return Candidate{
		SignalID: id,
		Name:     name,
		Kind:     "advanced",
		Buy:      buy,
		Sell:     sell,
	}, nil
}

func (c Catalog) findByID(id string) *Candidate {
	id = strings.TrimSpace(id)
	for _, slice := range [][]Candidate{c.Combinations, c.Indices, c.Advanced} {
		for i := range slice {
			if slice[i].SignalID == id {
				cp := slice[i]
				return &cp
			}
		}
	}
	return nil
}
