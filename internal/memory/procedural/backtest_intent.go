package procedural

import "strings"

const backtestRunPlaybook = "strategy-backtest-run"

// Explicit backtest verbs only — avoid hijacking generic analysis chat
// (e.g. "验证策略文档", "看收益率", "smarttrade 是什么").
var backtestRunIntentTokens = []string{
	"回测", "来回测", "跑回测", "backtest",
}

var backtestContinueIntentTokens = []string{
	"用现成的来回测", "不要新建", "继续回测", "再回测", "再跑回测", "沿用上次回测", "沿用那套回测",
	"就用刚才那套", "刚才那套", "沿用上次", "沿用那套",
}

// BacktestRunIntent reports whether the user wants PnL backtest rather than signal-only probe.
func BacktestRunIntent(message string) bool {
	if BacktestContinueIntent(message) {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(message))
	if msg == "" {
		return false
	}
	for _, tok := range backtestRunIntentTokens {
		if strings.Contains(msg, strings.ToLower(tok)) {
			return true
		}
	}
	return false
}

// BacktestContinueIntent reports follow-up turns that reuse an existing backtest setup.
func BacktestContinueIntent(message string) bool {
	msg := strings.ToLower(strings.TrimSpace(message))
	if msg == "" {
		return false
	}
	for _, tok := range backtestContinueIntentTokens {
		if strings.Contains(msg, tok) {
			return true
		}
	}
	return false
}

// ShouldBlockLegacyBacktestTools reports when ReAct must not pick DCA/grid loopback tools.
func ShouldBlockLegacyBacktestTools(message string) bool {
	if BacktestDCABypass(message) {
		return false
	}
	return BacktestRunIntent(message)
}

// BacktestDCABypass reports explicit DCA/grid/loopback intent that should stay on the legacy path.
func BacktestDCABypass(message string) bool {
	msg := strings.ToLower(strings.TrimSpace(message))
	for _, tok := range []string{"dca", "定投", "网格", "grid", "loopback", "generate_dca"} {
		if strings.Contains(msg, tok) {
			return true
		}
	}
	return false
}

// FindByName returns a loaded skill by name.
func (l *Loader) FindByName(name string) (Skill, bool) {
	if l == nil || strings.TrimSpace(name) == "" {
		return Skill{}, false
	}
	l.maybeRefresh()
	for _, sk := range l.skills {
		if sk.Name == name {
			return sk, true
		}
	}
	return Skill{}, false
}

// PrioritizeSignalProbePlaybook keeps strategy-signal-probe and parent strategy-backtest
// when the user wants signal-only probe; drops strategy-backtest-run from the matched set.
func (l *Loader) PrioritizeSignalProbePlaybook(message string, matched []Skill, maxSkills int) []Skill {
	if l == nil || !SignalProbeIntent(message) {
		return matched
	}
	if maxSkills <= 0 {
		maxSkills = 2
	}
	byName := map[string]Skill{}
	for _, sk := range matched {
		byName[sk.Name] = sk
	}
	if probe, ok := l.FindByName("strategy-signal-probe"); ok {
		byName[probe.Name] = probe
	}
	if parent, ok := l.FindByName("strategy-backtest"); ok {
		byName[parent.Name] = parent
	}
	priority := []string{"strategy-signal-probe", "strategy-backtest"}
	out := make([]Skill, 0, maxSkills)
	seen := map[string]struct{}{}
	for _, name := range priority {
		sk, ok := byName[name]
		if !ok {
			continue
		}
		out = append(out, sk)
		seen[name] = struct{}{}
		if len(out) >= maxSkills {
			return out
		}
	}
	var rest []Skill
	for _, sk := range matched {
		if _, ok := seen[sk.Name]; ok {
			continue
		}
		if sk.Name == backtestRunPlaybook {
			continue
		}
		rest = append(rest, sk)
	}
	for len(out) < maxSkills && len(rest) > 0 {
		out = append(out, rest[0])
		rest = rest[1:]
	}
	if len(out) > maxSkills {
		out = out[:maxSkills]
	}
	return out
}

// ShouldBlockSmartTradeBacktestTools reports when ReAct must not pick run_strategy_backtest.
func ShouldBlockSmartTradeBacktestTools(message string) bool {
	return SignalProbeIntent(message)
}

// PrioritizeBacktestRunPlaybook keeps strategy-backtest-run in the matched set when the user asks to run backtests.
func (l *Loader) PrioritizeBacktestRunPlaybook(message string, matched []Skill, maxSkills int) []Skill {
	if l == nil || !BacktestRunIntent(message) {
		return matched
	}
	if maxSkills <= 0 {
		maxSkills = 2
	}
	run, ok := l.FindByName(backtestRunPlaybook)
	if !ok {
		return matched
	}
	for _, sk := range matched {
		if sk.Name == backtestRunPlaybook {
			return matched
		}
	}
	byName := map[string]Skill{run.Name: run}
	for _, sk := range matched {
		byName[sk.Name] = sk
	}
	priority := []string{
		backtestRunPlaybook,
		"strategy-backtest",
		"strategy-backtest-history",
	}
	out := make([]Skill, 0, maxSkills)
	seen := map[string]struct{}{}
	for _, name := range priority {
		sk, ok := byName[name]
		if !ok {
			continue
		}
		out = append(out, sk)
		seen[name] = struct{}{}
		if len(out) >= maxSkills {
			return out
		}
	}
	var probe Skill
	hasProbe := false
	var rest []Skill
	for _, sk := range matched {
		if _, ok := seen[sk.Name]; ok {
			continue
		}
		if sk.Name == "strategy-signal-probe" {
			probe = sk
			hasProbe = true
			continue
		}
		rest = append(rest, sk)
	}
	for len(out) < maxSkills && len(rest) > 0 {
		out = append(out, rest[0])
		rest = rest[1:]
	}
	if len(out) < maxSkills && hasProbe {
		out = append(out, probe)
	}
	if len(out) > maxSkills {
		out = out[:maxSkills]
	}
	return out
}
