package feishu

import (
	"bytes"
	"testing"
)

func TestRenderQR(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderQR(&buf, "https://example.test/scan?from=geegoo"); err != nil {
		t.Fatal(err)
	}
	if buf.Len() < 50 {
		t.Fatalf("QR output too short: %d", buf.Len())
	}
}
