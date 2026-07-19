package tools_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestValidateArgumentsRequired(t *testing.T) {
	t.Parallel()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task": map[string]any{"type": "string"},
		},
		"required": []any{"task"},
	}
	if err := tools.ValidateArguments(schema, map[string]any{}); err == nil {
		t.Fatal("expected missing task error")
	}
	if err := tools.ValidateArguments(schema, map[string]any{"task": "x"}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestValidateArgumentsTypes(t *testing.T) {
	t.Parallel()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"max_steps": map[string]any{"type": "integer"},
		},
	}
	if err := tools.ValidateArguments(schema, map[string]any{"max_steps": "bad"}); err == nil {
		t.Fatal("expected type error")
	}
	if err := tools.ValidateArguments(schema, map[string]any{"max_steps": float64(3)}); err != nil {
		t.Fatalf("float64 int ok: %v", err)
	}
}

func TestValidateArgumentsEnum(t *testing.T) {
	t.Parallel()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"market": map[string]any{"type": "string", "enum": []any{"US", "HK", "CN"}},
		},
	}
	if err := tools.ValidateArguments(schema, map[string]any{"market": "XX"}); err == nil {
		t.Fatal("expected enum error")
	}
}

func TestValidateArgumentsNestedObject(t *testing.T) {
	t.Parallel()
	schema := map[string]any{
		"type": "object",
		"required": []any{"signal"},
		"properties": map[string]any{
			"signal": map[string]any{
				"type":     "object",
				"required": []any{"buy_signal"},
				"properties": map[string]any{
					"buy_signal": map[string]any{
						"type":     "array",
						"minItems": float64(1),
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"index": map[string]any{"type": "string"},
							},
						},
					},
				},
			},
		},
	}
	if err := tools.ValidateArguments(schema, map[string]any{
		"signal": map[string]any{"buy_signal": []any{}},
	}); err == nil {
		t.Fatal("expected minItems error")
	}
	if err := tools.ValidateArguments(schema, map[string]any{
		"signal": map[string]any{},
	}); err == nil || !strings.Contains(err.Error(), "signal.buy_signal") {
		t.Fatalf("expected nested required error, got %v", err)
	}
	if err := tools.ValidateArguments(schema, map[string]any{
		"signal": map[string]any{"buy_signal": []any{map[string]any{"index": 1}}},
	}); err == nil || !strings.Contains(err.Error(), "应为字符串") {
		t.Fatalf("expected nested type error, got %v", err)
	}
}

func TestValidateArgumentsMinItems(t *testing.T) {
	t.Parallel()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tasks": map[string]any{"type": "array", "minItems": float64(1)},
		},
	}
	if err := tools.ValidateArguments(schema, map[string]any{"tasks": []any{}}); err == nil {
		t.Fatal("expected minItems error")
	}
}

func TestRegistryRejectsInvalidArgs(t *testing.T) {
	t.Parallel()
	r := tools.NewRegistry()
	r.Register(tools.Tool{
		Name: "echo",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task": map[string]any{"type": "string"},
			},
			"required": []any{"task"},
		},
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			return tools.Result{Status: tools.StatusOK, Summary: "ok"}
		},
	})
	res := r.Execute(tools.CallRequest{Name: "echo", Arguments: map[string]any{}}, tools.Context{})
	if res.Status != tools.StatusError || !strings.Contains(res.Summary, "参数校验失败") {
		t.Fatalf("result=%+v", res)
	}
}
