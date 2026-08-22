package runtime_test

import (
	"encoding/json"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/runtime/events"
)

func TestProgressPayloadLegacyAndStructured(t *testing.T) {
	t.Parallel()
	payload := runtime.ProgressPayload("gate", map[string]any{
		"decision": "retrieve",
		"hits":     2,
	})
	if payload["schema_version"] != 1 {
		t.Fatalf("schema_version=%v", payload["schema_version"])
	}
	if payload["item_type"] != events.ItemStatus {
		t.Fatalf("item_type=%v", payload["item_type"])
	}
	if payload["decision"] != "retrieve" {
		t.Fatal("legacy flat decision missing")
	}
	nested, ok := payload["data"].(map[string]any)
	if !ok || nested["hits"] != 2 {
		t.Fatal("nested data missing")
	}
	raw, err := json.Marshal(payload)
	if err != nil || len(raw) == 0 {
		t.Fatal(err)
	}
}
