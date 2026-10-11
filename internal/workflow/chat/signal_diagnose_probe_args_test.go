package chat

import "fmt"

func probeArgsArePriceAnalysisFetch(args map[string]any) bool {
	buy, _ := args["buy_signal"].([]any)
	if len(buy) != 1 {
		return false
	}
	row, ok := buy[0].(map[string]any)
	if !ok {
		return false
	}
	return fmt.Sprint(row["index"]) == "nosignal"
}
