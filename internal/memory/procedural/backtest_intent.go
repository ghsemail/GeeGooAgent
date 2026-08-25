package procedural

import "strings"

const backtestRunPlaybook = "strategy-backtest-run"

var backtestRunIntentTokens = []string{
	"回测", "来回测", "跑回测", "看收益", "收益率", "回撤", "成交笔数", "验证策略",
	"backtest", "smarttrade",
}

var backtestContinueIntentTokens = []string{
	"用现成的", "不要新建", "继续回测", "再回测", "再跑", "同样", "上次", "刚才",
	"原有的", "现有的", "那套", "沿用",
}

// BacktestRunIntent reports whether the user wants PnL backtest rather than signal-only probe.
func BacktestRunIntent(message string) bool {
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
	return BacktestRunIntent(message) || BacktestContinueIntent(message)
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
