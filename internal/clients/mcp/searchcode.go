package mcp

func parseSearchCodeItem(m map[string]any) SearchCodeItem {
	code, _ := m["code"].(string)
	market, _ := m["market"].(string)
	stockType, _ := m["stock_type"].(string)
	lotSize := 0
	switch v := m["lot_size"].(type) {
	case float64:
		lotSize = int(v)
	case int:
		lotSize = v
	}

	nameEN, nameZH, nameHK := "", "", ""
	switch nameRaw := m["name"].(type) {
	case string:
		if nameRaw != "" {
			nameEN = nameRaw
		}
	case map[string]any:
		nameZH, _ = nameRaw["init"].(string)
		nameEN, _ = nameRaw["en"].(string)
		nameHK, _ = nameRaw["zh_hk"].(string)
		if nameHK == "" {
			nameHK, _ = nameRaw["ch_hk"].(string)
		}
	}
	display := nameZH
	if display == "" {
		display = nameEN
	}
	if display == "" {
		display = nameHK
	}
	return SearchCodeItem{
		Code: code, Name: display, NameEN: nameEN, NameZH: nameZH,
		Market: market, StockType: stockType, LotSize: lotSize,
	}
}
