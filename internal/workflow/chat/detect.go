package chat

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

var generateStrategyCognitionPhrases = []string{
	"生成策略档案", "策略档案",
	"生成策略认知", "策略认知",
	"学习策略", "了解策略", "整理策略", "写入知识库", "存入知识库",
}

var strategyDevPhrases = []string{
	"策略开发",
}

var signalDiagnosePhrases = []string{
	"信号诊断",
}

func isReadStrategyDevIntent(text string) bool {
	trim := strings.TrimSpace(text)
	if trim == "" {
		return false
	}
	if readStrategyDevMessagePattern.MatchString(trim) {
		return true
	}
	return strings.Contains(trim, "读取") && strings.Contains(trim, "策略")
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

// IsGenerateStrategyCognitionIntent reports generate_strategy_cognition workflow entry.
func IsGenerateStrategyCognitionIntent(text string) bool {
	trim := strings.TrimSpace(text)
	if trim == "" {
		return false
	}
	for _, p := range generateStrategyCognitionPhrases {
		if strings.Contains(trim, p) {
			return true
		}
	}
	return false
}

// IsSignalDiagnoseIntent reports signal_diagnose workflow entry.
func IsSignalDiagnoseIntent(text string) bool {
	trim := strings.TrimSpace(text)
	if trim == "" {
		return false
	}
	if signalDiagnoseMessagePattern.MatchString(trim) {
		return true
	}
	for _, p := range signalDiagnosePhrases {
		if strings.Contains(trim, p) {
			return true
		}
	}
	return strings.Contains(trim, "诊断") && (strings.Contains(trim, "策略") || strings.Contains(trim, "信号"))
}

// IsStrategyDevIntent reports strategy_dev workflow entry (read cognition from KB).
func IsStrategyDevIntent(text string) bool {
	trim := strings.TrimSpace(text)
	if trim == "" {
		return false
	}
	for _, p := range strategyDevPhrases {
		if strings.Contains(trim, p) {
			return true
		}
	}
	return isReadStrategyDevIntent(trim)
}

// ShouldStartGenerateStrategyCognitionFlow decides whether to create generate workflow.
func ShouldStartGenerateStrategyCognitionFlow(text string, existing *Flow) bool {
	if existing != nil && existing.Active() {
		return false
	}
	return IsGenerateStrategyCognitionIntent(text)
}

// ShouldStartStrategyDevFlow decides whether to create strategy_dev workflow.
func ShouldStartStrategyDevFlow(text string, existing *Flow) bool {
	if existing != nil && existing.Active() {
		return false
	}
	if IsSignalDiagnoseIntent(text) {
		return false
	}
	return IsStrategyDevIntent(text)
}

// ShouldStartSignalDiagnoseFlow decides whether to create signal_diagnose workflow.
func ShouldStartSignalDiagnoseFlow(text string, existing *Flow) bool {
	if !IsSignalDiagnoseIntent(text) {
		return false
	}
	if existing == nil || !existing.Active() {
		return true
	}
	switch existing.Status {
	case StatusPausedFailed, StatusInterrupted:
		return true
	}
	if existing.Template != SkillSignalDiagnose {
		return true
	}
	return false
}

// ShouldStartMultiStrategyFlow decides whether to create a new flow this turn.
func ShouldStartMultiStrategyFlow(text string, session *runtime.Session, existing *Flow) bool {
	if existing != nil && existing.Active() {
		return false
	}
	if IsGenerateStrategyCognitionIntent(text) || IsStrategyDevIntent(text) || IsSignalDiagnoseIntent(text) {
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
