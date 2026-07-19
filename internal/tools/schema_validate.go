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
		items, ok := value.([]any)
		if !ok {
			return fmt.Errorf("参数 %q 应为数组", field)
		}
		if min, ok := prop["minItems"].(float64); ok && float64(len(items)) < min {
			return fmt.Errorf("参数 %q 至少需要 %d 项", field, int(min))
		}
		if itemSchema, ok := prop["items"].(map[string]any); ok {
			for i, item := range items {
				if err := validateValue(fmt.Sprintf("%s[%d]", field, i), itemSchema, item); err != nil {
					return err
				}
			}
		}
	case "object":
		obj, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("参数 %q 应为对象", field)
		}
		if err := validateObjectFields(field, prop, obj); err != nil {
			return err
		}
	}
	if typ == "string" || typ == "" {
		if enum := stringSlice(prop["enum"]); len(enum) > 0 {
			s, ok := value.(string)
			if !ok || !stringIn(enum, s) {
				return fmt.Errorf("参数 %q 必须是 %v 之一", field, enum)
			}
		}
	}
	return nil
}

func validateObjectFields(prefix string, schema map[string]any, obj map[string]any) error {
	props, _ := schema["properties"].(map[string]any)
	required := stringSlice(schema["required"])
	for _, key := range required {
		field := key
		if prefix != "" {
			field = prefix + "." + key
		}
		v, ok := obj[key]
		if !ok {
			return fmt.Errorf("缺少必填参数 %q", field)
		}
		if isEmptyArg(v) {
			return fmt.Errorf("参数 %q 不能为空", field)
		}
	}
	if props == nil {
		return nil
	}
	for key, raw := range obj {
		prop, ok := props[key].(map[string]any)
		if !ok {
			continue
		}
		field := key
		if prefix != "" {
			field = prefix + "." + key
		}
		if err := validateValue(field, prop, raw); err != nil {
			return err
		}
	}
	return nil
}

func stringIn(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
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
