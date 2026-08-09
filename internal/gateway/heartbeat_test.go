package gateway

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHeartbeatRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	// UserHomeDir on Windows uses USERPROFILE
	t.Setenv("USERPROFILE", dir)

	path := HeartbeatFile()
	if filepath.Base(path) != "gateway.heartbeat.json" {
		t.Fatalf("path=%s", path)
	}
	if err := WriteHeartbeat(HeartbeatSnapshot{
		Platform:  "feishu",
		Connected: true,
		Detail:    "ok",
	}); err != nil {
		t.Fatal(err)
	}
	snap, ok := ReadHeartbeat()
	if !ok || !snap.Connected || snap.Platform != "feishu" {
		t.Fatalf("snap=%+v ok=%v", snap, ok)
	}
	if !HeartbeatFresh(snap, time.Minute) {
		t.Fatal("expected fresh")
	}
	snap.UpdatedAt = time.Now().UTC().Add(-2 * time.Minute)
	if HeartbeatFresh(snap, time.Minute) {
		t.Fatal("expected stale")
	}
	_ = os.Remove(path)
}
