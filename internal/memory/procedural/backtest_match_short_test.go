package procedural

import "testing"

func TestMatchShortBacktestReuseMessage(t *testing.T) {
	loader := NewLoader("../../../skills")
	msg := "用现成的来回测，不要新建"
	matched := loader.Match(msg, 2)
	names := make([]string, len(matched))
	for i, sk := range matched {
		names[i] = sk.Name
	}
	t.Logf("matched=%v intent=%v", names, BacktestRunIntent(msg))
	if len(matched) == 0 {
		t.Log("no skills matched - playbookexec Route will fail")
	}
}
