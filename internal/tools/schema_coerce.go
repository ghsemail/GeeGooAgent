package tools

import (
	"encoding/json"
	"strconv"
	"strings"
)

// CoerceArguments normalizes LLM tool args to schema types before validation.
func CoerceArguments(schema map[string]any, args map[string]any) {
	if schema == nil || args == nil {
		return
	}
	props, _ := schema["properties"].(map[string]any)
	if props == nil {
		return
	}
	for key, raw := range args {
		prop, ok := props[key].(map[string]any)
		if !ok {
			continue
		}
		args[key] = coerceSchemaValue(prop, raw)
	}
}

func coerceSchemaValue(prop map[string]any, value any) any {
	if value == nil {
		return nil
	}
	typ, _ := prop["type"].(string)
	switch typ {
	case "integer":
		if i, ok := coerceInteger(value); ok {
			return i
		}
	case "number":
		if f, ok := coerceNumber(value); ok {
			return f
		}
	case "array":
		if arr, ok := coerceJSONArray(prop, value); ok {
			return arr
		}
	case "object":
		if obj, ok := coerceJSONObject(value); ok {
			return obj
		}
	}
	return value
}

func coerceInteger(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float64:
		return int(n), n == float64(int(n))
	case float32:
		return int(n), float32(int(n)) == n
	case string:
		s := strings.TrimSpace(n)
		if s == "" {
			return 0, false
		}
		i, err := strconv.Atoi(s)
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}

func coerceNumber(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case string:
		s := strings.TrimSpace(n)
		if s == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

func coerceJSONArray(prop map[string]any, value any) ([]any, bool) {
	if arr, ok := value.([]any); ok {
		itemSchema, _ := prop["items"].(map[string]any)
		if itemSchema == nil {
			return arr, true
		}
		out := make([]any, len(arr))
		for i, item := range arr {
			out[i] = coerceSchemaValue(itemSchema, item)
		}
		return out, true
	}
	s, ok := value.(string)
	if !ok {
		return nil, false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, false
	}
	var arr []any
	if err := json.Unmarshal([]byte(s), &arr); err != nil {
		return nil, false
	}
	return coerceJSONArray(prop, arr)
}

func coerceJSONObject(value any) (map[string]any, bool) {
	if obj, ok := value.(map[string]any); ok {
		return obj, true
	}
	s, ok := value.(string)
	if !ok {
		return nil, false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, false
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		return nil, false
	}
	return obj, true
}
