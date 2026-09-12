package catalog

// ApplyProbeDefaults fills probe timing defaults and normalizes integer fields.
func ApplyProbeDefaults(body map[string]any) {
	if body == nil {
		return
	}
	if months, ok := IntFromAny(body["months_back"]); ok && months > 0 {
		body["months_back"] = months
	} else if limit, ok := IntFromAny(body["limit"]); ok && limit > 0 {
		body["limit"] = limit
	} else {
		body["months_back"] = 3
	}
}
