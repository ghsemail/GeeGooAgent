package tools

import (
	"fmt"
	"strings"
)

// ValidateArguments checks args against a JSON-schema-like Parameters map.
// Returns a short, LLM-friendly error when validation fails.
func ValidateArguments(schema map[string]any, args map[string]any) error {
	if schema == nil {
		return nil
	}
	if args == nil {
		args = map[string]any{}
	}
	props, _ := schema["properties"].(map[string]any)
	required := stringSlice(schema["required"])
	for _, key := range required {
		if _, ok := args[key]; !ok {
			return fmt.Errorf("缺少必填参数 %q", key)
		}
		if isEmptyArg(args[key]) {
			return fmt.Errorf("参数 %q 不能为空", key)
		}
	}
	if props == nil {
		return nil
	}
	for key, raw := range args {
		prop, ok := props[key].(map[string]any)
		if !ok {
			continue
		}
		if err := validateValue(key, prop, raw); err != nil {
			return err
		}
	}
	return nil
}

func validateValue(field string, prop map[string]any, value any) error {
	if value == nil {
		return nil
	}
	typ, _ := prop["type"].(string)
	if typ == "" {
		return nil
	}
	switch typ {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("参数 %q 应为字符串", field)
		}
	case "integer":
		if !isInteger(value) {
			return fmt.Errorf("参数 %q 应为整数", field)
		}
	case "number":
		if !isNumber(value) {
			return fmt.Errorf("参数 %q 应为数字", field)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("参数 %q 应为布尔值", field)
		}
	case "array":
		if _, ok := value.([]any); !ok {
			return fmt.Errorf("参数 %q 应为数组", field)
		}
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return fmt.Errorf("参数 %q 应为对象", field)
		}
	}
	return nil
}

func isEmptyArg(v any) bool {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t) == ""
	case nil:
		return true
	default:
		return false
	}
}

func isInteger(v any) bool {
	switch n := v.(type) {
	case int, int32, int64:
		return true
	case float64:
		return n == float64(int64(n))
	default:
		return false
	}
}

func isNumber(v any) bool {
	switch v.(type) {
	case int, int32, int64, float32, float64:
		return true
	default:
		return false
	}
}

func stringSlice(raw any) []string {
	items, ok := raw.([]any)
	if !ok {
		if typed, ok := raw.([]string); ok {
			return typed
		}
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		s, _ := item.(string)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
