package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
	"github.com/ghsemail/GeeGooAgent/internal/lineage"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// UpstreamMessage is one OpenAI-style chat message from GeeGooBot agent-api.
type UpstreamMessage struct {
	Role    string
	Content string
}

// Session holds in-memory chat state for one conversation.
type Session struct {
	ID               string
	Messages         []llm.Message
	PreviousSummary  string
	LastPromptTokens int
	StepCounter      int
	CreatedAt        time.Time
	// Lineage tracks Hermes-style compaction bloodline without forking the
	// user-facing session id. ParentID is the previous generation node;
	// LineageRoot is the original session id before any compaction.
	ParentID             string
	LineageRoot          string
	CompactionGeneration int
	// PendingPlan holds mutating tool_calls awaiting user confirmation (plan gate).
	PendingPlan *PendingPlan
	// LineageChain records each context compression/hygiene event (newest last).
	LineageChain []lineage.Record
	// UserID is the tenant owner for session recall scoping (from chat metadata).
	UserID string
	// LastTurnDomain is the cognition domain from the previous user turn.
	LastTurnDomain string
	// LastTurnMode is the cognition mode from the previous user turn.
	LastTurnMode string
	// LastTurnSOP mirrors ShouldRunDomainSOP for the previous user turn.
	LastTurnSOP bool
	// LastTurnToolsAllow is the tool whitelist from the previous turn plan.
	LastTurnToolsAllow []string
	// LastTurnAct is the catalog act from the previous turn plan.
	LastTurnAct string
	// PriorSessionTools is the union of tools from prior user turns (loaded from chat).
	PriorSessionTools []string
	// LastExecutionProfile is the profile id applied to the current/last turn.
	LastExecutionProfile string
	// ActiveFlowJSON holds opaque chat workflow state (see internal/workflow/chat).
	ActiveFlowJSON []byte
	// PendingSignalDiagnoseOpts applies to the next new signal_diagnose flow only (not persisted).
	PendingSignalDiagnoseOpts *SignalDiagnoseOpts
	// PendingStrategyDevOpts applies to the next strategy_dev_pipeline flow only (not persisted).
	PendingStrategyDevOpts *StrategyDevOpts
}

// StrategyDevOpts configures strategy_dev_pipeline for one chat turn.
type StrategyDevOpts struct {
	Code       string `json:"code,omitempty"`
	Frequency  string `json:"frequency,omitempty"`
	MonthsBack int    `json:"months_back,omitempty"`
	UsePriceAnalysisEpisodeFilter *bool    `json:"use_price_analysis_episode_filter,omitempty"`
	PriceAnalysisScoreThreshold   *float64 `json:"price_analysis_score_threshold,omitempty"`
}

// SignalDiagnoseOpts configures optional signal_diagnose workflow behavior for one turn.
type SignalDiagnoseOpts struct {
	// UsePriceAnalysisEpisodeFilter nil = default off (soft annotate); true = hard mask before episode eval.
	UsePriceAnalysisEpisodeFilter *bool `json:"use_price_analysis_episode_filter,omitempty"`
	// PriceAnalysisScoreThreshold nil = default 0 for hard gate buy/sell score cutoffs.
	PriceAnalysisScoreThreshold *float64 `json:"price_analysis_score_threshold,omitempty"`
	// PriceAnalysisBuyScoreDirection soft buy would_intercept: normal (score<threshold) | reverse (score>threshold).
	PriceAnalysisBuyScoreDirection string `json:"price_analysis_buy_score_direction,omitempty"`
	// PriceAnalysisGateScoreMode full | htf_only | tf_tiered (default full).
	PriceAnalysisGateScoreMode string `json:"price_analysis_gate_score_mode,omitempty"`
	// Optional probe overrides from strategy-dev panel (buy/sell configured separately).
	BuySignal   []any  `json:"buy_signal,omitempty"`
	SellSignal  []any  `json:"sell_signal,omitempty"`
	// SellMerge forwarded to probeBotSignalSeries: and (default) | or.
	SellMerge   string `json:"sell_merge,omitempty"`
	// BuyMerge optional; forwarded when set.
	BuyMerge    string `json:"buy_merge,omitempty"`
	Frequency   string `json:"frequency,omitempty"`
	MonthsBack  int    `json:"months_back,omitempty"`
	// PanelStrategyLabel is the strategy-dev UI title (may differ from chat utterance).
	PanelStrategyLabel string `json:"panel_strategy_label,omitempty"`
}

// PendingPlan is a held mutating-tool batch from one LLM round.
type PendingPlan struct {
	Step      int
	ToolCalls []llm.ToolCall
}

// NewSession creates a chat session with system prompt.
func NewSession() *Session {
	return &Session{
		ID: newSessionID(),
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: chatprompt.System()},
		},
		CreatedAt: time.Now().UTC(),
	}
}

func newSessionID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "chat-" + hex.EncodeToString(b[:])
}

// NewUpstreamSession builds a session from agent-api message history.
func NewUpstreamSession(messages []UpstreamMessage) (*Session, string) {
	session := &Session{
		ID:        newSessionID(),
		CreatedAt: time.Now().UTC(),
	}
	var lastUser string
	for _, m := range messages {
		role := llm.Role(m.Role)
		if role != llm.RoleSystem && role != llm.RoleUser && role != llm.RoleAssistant {
			continue
		}
		session.AppendMessage(llm.Message{Role: role, Content: m.Content})
		if role == llm.RoleUser {
			lastUser = m.Content
		}
	}
	if len(session.Messages) > 0 {
		last := session.Messages[len(session.Messages)-1]
		if last.Role == llm.RoleUser && last.Content == lastUser {
			session.Messages = session.Messages[:len(session.Messages)-1]
		}
	}
	if len(session.Messages) == 0 || session.Messages[0].Role != llm.RoleSystem {
		session.Messages = append([]llm.Message{{Role: llm.RoleSystem, Content: chatprompt.System()}}, session.Messages...)
	}
	return session, lastUser
}

// AppendMessage adds a message to the session.
func (s *Session) AppendMessage(m llm.Message) {
	s.Messages = append(s.Messages, m)
}

// LLMMessages returns a copy for the gateway.
func (s *Session) LLMMessages() []llm.Message {
	out := make([]llm.Message, len(s.Messages))
	copy(out, s.Messages)
	return out
}
