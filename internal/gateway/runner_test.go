package gateway

import "testing"

func TestDedupCache(t *testing.T) {
	d := newDedupCache(2)
	if !d.TryAdd("a") {
		t.Fatal("first a")
	}
	if d.TryAdd("a") {
		t.Fatal("dup a")
	}
	if !d.TryAdd("b") || !d.TryAdd("c") {
		t.Fatal("b/c")
	}
	// a should be evicted
	if !d.TryAdd("a") {
		t.Fatal("a after eviction")
	}
}

func TestAuthorize(t *testing.T) {
	r := &Runner{Config: Config{
		AllowedUsers: map[Platform]map[string]struct{}{
			PlatformFeishu: {"ou_ok": {}},
		},
		AllowAll: map[Platform]bool{PlatformFeishu: false},
	}}
	if !r.authorize(InboundEvent{Platform: PlatformFeishu, UserID: "ou_ok"}) {
		t.Fatal("ou_ok")
	}
	if r.authorize(InboundEvent{Platform: PlatformFeishu, UserID: "ou_no"}) {
		t.Fatal("ou_no")
	}
	r.Config.AllowAll[PlatformFeishu] = true
	if !r.authorize(InboundEvent{Platform: PlatformFeishu, UserID: "ou_no"}) {
		t.Fatal("allow all")
	}
}
