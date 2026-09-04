package slots

// CatalogItems normalizes tool payload list shapes (items/data/value).
func CatalogItems(data map[string]any) []map[string]any {
	if data == nil {
		return nil
	}
	if items, ok := data["items"].([]any); ok {
		return mapsFromAny(items)
	}
	if arr, ok := data["data"].([]any); ok {
		return mapsFromAny(arr)
	}
	if arr, ok := data["value"].([]any); ok {
		return mapsFromAny(arr)
	}
	return mapsFromAny([]any{data})
}

func mapsFromAny(items []any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, raw := range items {
		row, ok := raw.(map[string]any)
		if ok {
			out = append(out, row)
		}
	}
	return out
}
