package gateway

import "testing"

func TestMaskAppIDAndSecret(t *testing.T) {
	id := MaskAppID("cli_a1b2c3d4e5f6g7h8")
	if id[:4] != "cli_" || !containsStars(id) || id[len(id)-4:] != "g7h8" {
		t.Fatalf("app id mask=%q", id)
	}
	sec := MaskAppSecret("abcdefghijklmnopqrstuvwxyz")
	if sec[:3] != "abc" || !containsStars(sec) || sec[len(sec)-3:] != "xyz" {
		t.Fatalf("secret mask=%q", sec)
	}
	if MaskAppID("short") != "*****" {
		t.Fatalf("short=%q", MaskAppID("short"))
	}
}

func containsStars(s string) bool {
	for _, r := range s {
		if r == '*' {
			return true
		}
	}
	return false
}
