package chatprompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseProfileRef(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		kind ProfileKind
		key  string
		ok   bool
	}{
		{"market:HK", ProfileMarket, "HK", true},
		{"stock:00700.HK", ProfileStock, "00700.HK", true},
		{"automation:bot-1", ProfileAutomation, "bot-1", true},
		{"user_default", ProfileUserDefault, "", true},
		{"bad", ProfileKind(""), "", false},
		{"market:", ProfileKind(""), "", false},
	}
	for _, c := range cases {
		ref, err := ParseProfileRef(c.in)
		if c.ok && err != nil {
			t.Fatalf("%q: %v", c.in, err)
		}
		if !c.ok && err == nil {
			t.Fatalf("%q: expected error", c.in)
		}
		if c.ok {
			if ref.Kind != c.kind || ref.Key != c.key {
				t.Fatalf("%q => %+v", c.in, ref)
			}
		}
	}
}

func TestMergeProfilesOrderAndLimit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	uid := "user-1"
	write := func(rel, body string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("AGENTS.md", "global rules\n")
	write(filepath.Join("tenants", uid, "AGENTS.md"), "user rules\n")
	write(filepath.Join("tenants", uid, "markets", "HK", "AGENTS.md"), "hk market\n")
	write(filepath.Join("tenants", uid, "stocks", "00700.HK", "AGENTS.md"), "tencent stock\n")
	write(filepath.Join("tenants", uid, "automations", "b1", "AGENTS.md"), "bot b1\n")

	merge := MergeProfiles(dir, uid, []string{
		"automation:b1",
		"stock:00700.HK",
		"market:HK",
	}, ProfileLimits{MaxMergedBytes: 4096, MaxProfilesPerSession: 4})
	if !containsAll(merge.Text, "global rules", "user rules", "hk market", "tencent stock", "bot b1") {
		t.Fatalf("merge missing sections: %q", merge.Text)
	}
	idxMarket := indexOf(merge.Text, "hk market")
	idxStock := indexOf(merge.Text, "tencent stock")
	idxBot := indexOf(merge.Text, "bot b1")
	if !(idxMarket < idxStock && idxStock < idxBot) {
		t.Fatalf("wrong merge order: market@%d stock@%d bot@%d", idxMarket, idxStock, idxBot)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !contains(s, p) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	return strings.Index(s, sub)
}
