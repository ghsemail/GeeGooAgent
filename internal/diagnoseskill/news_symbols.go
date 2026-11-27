package diagnoseskill

import "strings"

const maxMacroNewsSymbols = 10

// MacroNewsSymbol is an instrument the item mentions. The caller normalizes the code
// and completes a missing name from its stock data.
type MacroNewsSymbol struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Market string `json:"market"`
}

// resolveSymbols keeps the model's symbols whose quoted mention really is in the
// title or snippet, so a pool's representative index is not attached to every story.
func resolveSymbols(raw any, in MacroNewsItem) []MacroNewsSymbol {
	out := []MacroNewsSymbol{}
	rows, _ := raw.([]any)
	text := strings.ToLower(in.Title + "\n" + in.Snippet)
	seen := map[string]bool{}
	for _, row := range rows {
		fields, _ := row.(map[string]any)
		symbol := strings.ToUpper(label(fields["symbol"]))
		mention := strings.ToLower(label(fields["mention"]))
		if symbol == "" || len(symbol) > 24 || mention == "" || !strings.Contains(text, mention) || seen[symbol] {
			continue
		}
		seen[symbol] = true
		out = append(out, MacroNewsSymbol{
			Symbol: symbol, Name: clipRunes(label(fields["name"]), 40),
			Market: strings.ToUpper(label(fields["market"])),
		})
		if len(out) == maxMacroNewsSymbols {
			break
		}
	}
	return out
}
