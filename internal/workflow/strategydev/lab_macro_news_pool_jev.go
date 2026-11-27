package strategydev

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// MacroNewsPoolItem is one analyzed headline entering the pool rollup.
type MacroNewsPoolItem struct {
	NewsID       string
	Title        string
	JEVDirection string // long | short | neutral
	JEVReason    string
}

// Above this many items each reason is shortened so the Choice state stays bounded.
const macroNewsCompactStateAt = 60

// ReasonStatus says whether a verdict's reason is a real summary.
const (
	MacroNewsReasonOK     = "ok"
	MacroNewsReasonFailed = "failed"
)

// MacroNewsVerdict is a pool or cross-pool verdict. With OK set, Direction is the
// Decision Choice's; ReasonStatus failed means only the summary is missing.
type MacroNewsVerdict struct {
	MacroPoolJEV
	ReasonStatus string
}

func truncateRunes(s string, n int) string {
	rs := []rune(strings.TrimSpace(s))
	if len(rs) <= n {
		return string(rs)
	}
	return string(rs[:n]) + "…"
}

// BuildMacroNewsPoolState builds the Decision Choice state from every item verdict.
func BuildMacroNewsPoolState(title string, items []MacroNewsPoolItem) string {
	reasonLen := 200
	if len(items) > macroNewsCompactStateAt {
		reasonLen = 60
	}
	var b strings.Builder
	if t := strings.TrimSpace(title); t != "" {
		fmt.Fprintf(&b, "监控池：%s\n", t)
	}
	for i, item := range items {
		fmt.Fprintf(&b, "\n【新闻 %d】%s\n", i+1, truncateRunes(item.Title, 160))
		fmt.Fprintf(&b, "新闻 JEV 方向：%s\n", ledgerDirectionToChoice(item.JEVDirection))
		if reason := strings.TrimSpace(item.JEVReason); reason != "" {
			fmt.Fprintf(&b, "依据摘要：%s\n", truncateRunes(reason, reasonLen))
		}
	}
	return strings.TrimSpace(b.String())
}

type macroNewsReasonHeadline struct {
	Title        string `json:"title"`
	JEVDirection string `json:"jev_direction"`
	Reason       string `json:"reason,omitempty"`
}

// chooseMacroNewsDirection asks Decision Choice for a direction. A failed or
// unusable call leaves OK unset: it is an error, never neutral.
func chooseMacroNewsDirection(
	ctx context.Context, jev *JevClient, state, instructions string, hasLong, hasShort bool,
) MacroNewsVerdict {
	model := config.DefaultDecisionModel
	if jev != nil && strings.TrimSpace(jev.cfg.Model) != "" {
		model = strings.TrimSpace(jev.cfg.Model)
	}
	out := MacroNewsVerdict{MacroPoolJEV: MacroPoolJEV{Model: model, Choices: priceAnalysisJEVChoicesFromProbs(nil)}}
	if jev == nil || strings.TrimSpace(jev.cfg.APIKey) == "" {
		out.Error = "jev not configured"
		return out
	}
	choice, probs, confidence, respModel, err := jev.Choice(ctx, state, "direction", instructions, macroNewsJEVCriteria)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	if respModel != "" {
		out.Model = respModel
	}
	idx, ok := indexPriceAnalysisJEVLabel(choice)
	if !ok {
		out.Error = fmt.Sprintf("unexpected jev choice: %q", choice)
		return out
	}
	out.Choices = priceAnalysisJEVChoicesFromProbs(probs)
	out.RecommendedChoice = choice
	conf := clampUnitInterval(probs[priceAnalysisJEVLabels[idx]])
	if confidence > 0 {
		conf = clampUnitInterval(confidence)
	}
	out.Conflict = hasLong && hasShort
	if out.Conflict && conf > labJEVUnifiedConflictConfidenceCap {
		conf = labJEVUnifiedConflictConfidenceCap
	}
	out.Confidence = conf
	out.Direction = choiceToLedgerDirection(choice)
	out.OK = true
	return out
}

// ResolveMacroNewsPoolJEV gets the pool direction from Decision Choice (never by
// counting item votes) and has the synthesis model write the reason. When no summary
// can be written the reason says so; no title list stands in for it. It reads news
// only.
func ResolveMacroNewsPoolJEV(
	ctx context.Context,
	jev *JevClient,
	poolID, poolLabel string,
	items []MacroNewsPoolItem,
	summarizer MacroNewsSummarizer,
) MacroNewsVerdict {
	if len(items) == 0 {
		return MacroNewsVerdict{MacroPoolJEV: MacroPoolJEV{
			Choices: priceAnalysisJEVChoicesFromProbs(nil), Error: "no analyzed news",
		}}
	}
	hasLong, hasShort := false, false
	for _, item := range items {
		hasLong = hasLong || item.JEVDirection == "long"
		hasShort = hasShort || item.JEVDirection == "short"
	}
	instructions := "基于 state 中该监控池当日高价值新闻的标题、JEV 方向与依据摘要，综合判断该池当日整体方向。" +
		macroNewsChoiceRules
	out := chooseMacroNewsDirection(ctx, jev, BuildMacroNewsPoolState(poolLabel, items), instructions, hasLong, hasShort)
	if !out.OK {
		return out
	}
	summary, err := summarizeMacroNewsPool(ctx, summarizer, poolID, poolLabel, out.MacroPoolJEV, items)
	out.setReason(summary, err)
	return out
}

func (v *MacroNewsVerdict) setReason(summary string, err error) {
	if err == nil {
		v.RecommendedReason, v.ReasonStatus = summary, MacroNewsReasonOK
		return
	}
	v.RecommendedReason, v.ReasonStatus = macroNewsSummaryFailure(err), MacroNewsReasonFailed
}
