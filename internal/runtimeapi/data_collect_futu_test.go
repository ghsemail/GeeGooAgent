package runtimeapi

import "testing"

func TestFutuHealthOK(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  map[string]any
		want bool
	}{
		{name: "ok field", raw: map[string]any{"ok": true}, want: true},
		{name: "code 100", raw: map[string]any{"code": 100, "message": "ok"}, want: true},
		{name: "message ok", raw: map[string]any{"message": "OK"}, want: true},
		{name: "failure", raw: map[string]any{"code": 502}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := futuHealthOK(tt.raw); got != tt.want {
				t.Fatalf("futuHealthOK() = %v, want %v", got, tt.want)
			}
		})
	}
}
