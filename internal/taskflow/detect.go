package taskflow

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/sessiontask"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

var continueAllPhrases = []string{
	"挨个跑", "挨个测", "挨个试", "依次跑", "依次测", "逐个跑", "逐个测",
	"都跑一下", "都测一下", "全部跑", "全部测", "一个一个跑", "一个一个测",
	"继续跑", "接着跑", "跑剩下的", "测剩下的",
}

var multiStrategyPhrases = []string{
	"多策略", "几个策略", "多种策略", "哪个信号多", "信号对比", "策略对比",
	"对比信号", "对比策略", "vs", "VS",
}

var resumePhrases = []string{
	"继续", "重试失败", "重试", "retry", "resume", "恢复",
}

var cancelPhrases = []string{
	"取消任务", "停止任务", "取消 flow", "停止 flow",
}

// IsCancelIntent reports explicit flow cancellation.
func IsCancelIntent(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	for _, p := range cancelPhrases {
		if strings.Contains(lower, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

// IsResumeIntent reports user wants to continue or retry a paused flow.
func IsResumeIntent(text string) bool {
	trim := strings.TrimSpace(text)
	if trim == "" {
		return false
	}
	lower := strings.ToLower(trim)
	for _, p := range resumePhrases {
		if lower == strings.ToLower(p) || strings.HasPrefix(lower, strings.ToLower(p)) {
			return true
		}
	}
	return IsContinueAllIntent(text)
}

// IsContinueAllIntent reports batch-run phrasing like「挨个跑一下」.
func IsContinueAllIntent(text string) bool {
	trim := strings.TrimSpace(text)
	if trim == "" {
		return false
	}
	for _, p := range continueAllPhrases {
		if strings.Contains(trim, p) {
			return true
		}
	}
	return false
}

// IsMultiStrategyCompareIntent reports a new multi-strategy probe batch on one symbol.
func IsMultiStrategyCompareIntent(text string, session *runtime.Session) bool {
	trim := strings.TrimSpace(text)
	if trim == "" {
		return false
	}
	if IsContinueAllIntent(trim) {
		st := sessiontask.BuildState(session, "signal_probe")
		if st.Symbol != "" || slots.ExtractExplicitStockReference(trim) != "" {
			return true
		}
		return sessionHasStockMention(session)
	}
	for _, p := range multiStrategyPhrases {
		if strings.Contains(trim, p) {
			return true
		}
	}
	if countStrategyMentions(trim) >= 2 {
		return true
	}
	return false
}

// ShouldStartMultiStrategyFlow decides whether to create a new flow this turn.
func ShouldStartMultiStrategyFlow(text string, session *runtime.Session, existing *Flow) bool {
	if existing != nil && existing.Active() {
		return false
	}
	return IsMultiStrategyCompareIntent(text, session)
}

// ShouldHandleFlowTurn decides whether taskflow should take over this turn.
func ShouldHandleFlowTurn(text string, flow *Flow) bool {
	if flow == nil {
		return false
	}
	if flow.Active() {
		return true
	}
	return IsResumeIntent(text) && flow.Status == StatusPausedFailed
}

func sessionHasStockMention(session *runtime.Session) bool {
	if session == nil {
		return false
	}
	for _, msg := range session.LLMMessages() {
		if slots.ExtractExplicitStockReference(msg.Content) != "" {
			return true
		}
	}
	return false
}

func countStrategyMentions(text string) int {
	upper := strings.ToUpper(text)
	keys := []string{"MACD", "SAR", "RSI", "共振", "4H", "节奏", "BBAND", "KDJ"}
	n := 0
	for _, k := range keys {
		if strings.Contains(upper, strings.ToUpper(k)) || strings.Contains(text, k) {
			n++
		}
	}
	return n
}
