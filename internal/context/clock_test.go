package context

import (
	"strings"
	"testing"
	"time"
)

func TestClockFragmentRendersShanghaiTime(t *testing.T) {
	t.Parallel()
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 8, 24, 14, 50, 0, 0, loc)

	frag := ClockFragment(now)
	if frag.Kind() != KindClock {
		t.Fatalf("kind=%q want %q", frag.Kind(), KindClock)
	}
	text := frag.Render()
	for _, want := range []string{
		"当前时间：2026-08-24 14:50",
		"星期一",
		"Asia/Shanghai",
		"今天",
		"今早",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
	if frag.Priority() != 1 {
		t.Fatalf("priority=%d want 1 (must survive compression)", frag.Priority())
	}
}

func TestClockFragmentConvertsUTCToShanghai(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 24, 6, 50, 0, 0, time.UTC) // 14:50 Asia/Shanghai
	text := ClockFragment(now).Render()
	if !strings.Contains(text, "2026-08-24 14:50") {
		t.Fatalf("expected Shanghai wall clock, got:\n%s", text)
	}
}

func TestRegisteredKindsIncludesClock(t *testing.T) {
	t.Parallel()
	found := false
	for _, k := range RegisteredKinds() {
		if k == KindClock {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("RegisteredKinds missing KindClock: %v", RegisteredKinds())
	}
}
