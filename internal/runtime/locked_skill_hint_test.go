package runtime

import "testing"

func TestLockedSkillTurnHint(t *testing.T) {
	h := LockedSkillTurnHint("signal_diagnose")
	if h == "" || !containsAll(h, "locked_skill", "signal_diagnose", "skill_view") {
		t.Fatalf("hint: %q", h)
	}
	if LockedSkillTurnHint("") != "" {
		t.Fatal("empty locked should yield empty hint")
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if sub == "" {
			continue
		}
		if !containsSub(s, sub) {
			return false
		}
	}
	return true
}

func containsSub(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && searchSub(s, sub))
}

func searchSub(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
